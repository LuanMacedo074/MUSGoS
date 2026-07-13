package outbound

import (
	"database/sql"
	"fmt"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// sqlDB is the storage core shared by the SQL-backed adapters (RFC-007). It
// owns the connection pool and defers backend differences to its dialect; the
// duplicated method bodies in the sqlite/postgres adapters migrate here group
// by group, written once with ?-placeholders and rebound per dialect.
type sqlDB struct {
	db      *sql.DB
	dialect dialect
}

// ensureMigrationsTable bootstraps the table the MigrationTracker records
// applied migrations in; called from the adapters' constructors.
func (d *sqlDB) ensureMigrationsTable() error {
	_, err := d.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS migrations (
			id %s,
			name TEXT NOT NULL UNIQUE,
			applied_at %s NOT NULL
		)
	`, d.dialect.AutoIncrPKSQL(), d.dialect.ColumnType(ports.ColDatetime)))
	return err
}

// --- MigrationTracker ---

func (d *sqlDB) GetAppliedMigrations() ([]string, error) {
	rows, err := d.db.Query("SELECT name FROM migrations ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (d *sqlDB) MarkMigrationApplied(name string) error {
	_, err := d.db.Exec(d.dialect.Rebind("INSERT INTO migrations (name, applied_at) VALUES (?, ?)"), name, time.Now())
	return err
}

// --- DBAdmin ---

func (d *sqlDB) CreateApplication(appName string) error {
	_, err := d.db.Exec(d.dialect.Rebind("INSERT INTO applications (name) VALUES (?)"), appName)
	return err
}

func (d *sqlDB) DeleteApplication(appName string) error {
	result, err := d.db.Exec(d.dialect.Rebind("DELETE FROM applications WHERE name = ?"), appName)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("application %q not found", appName)
	}
	return nil
}

func (d *sqlDB) getAppID(appName string) (int64, error) {
	var id int64
	err := d.db.QueryRow(d.dialect.Rebind("SELECT id FROM applications WHERE name = ?"), appName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("application %q not found: %w", appName, err)
	}
	return id, nil
}

// --- DBApplication ---

func (d *sqlDB) SetApplicationAttribute(appName, attrName string, value lingo.LValue) error {
	appID, err := d.getAppID(appName)
	if err != nil {
		return err
	}

	jsonBytes, err := lingo.MarshalLValue(value)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(d.dialect.Rebind(`
		INSERT INTO application_attributes (app_id, attr_name, value_json)
		VALUES (?, ?, ?)
		ON CONFLICT(app_id, attr_name) DO UPDATE SET value_json=excluded.value_json`),
		appID, attrName, string(jsonBytes))
	return err
}

func (d *sqlDB) GetApplicationAttribute(appName, attrName string) (lingo.LValue, error) {
	appID, err := d.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return d.scanAttribute(
		d.dialect.Rebind("SELECT value_json FROM application_attributes WHERE app_id = ? AND attr_name = ?"),
		appID, attrName)
}

func (d *sqlDB) GetApplicationAttributeNames(appName string) ([]string, error) {
	appID, err := d.getAppID(appName)
	if err != nil {
		return nil, err
	}
	return d.queryNames(d.dialect.Rebind("SELECT attr_name FROM application_attributes WHERE app_id = ?"), appID)
}

func (d *sqlDB) DeleteApplicationAttribute(appName, attrName string) error {
	appID, err := d.getAppID(appName)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(d.dialect.Rebind("DELETE FROM application_attributes WHERE app_id = ? AND attr_name = ?"), appID, attrName)
	return err
}

// --- DBPlayer ---

func (d *sqlDB) SetPlayerAttribute(appName, userID, attrName string, value lingo.LValue) error {
	appID, err := d.getAppID(appName)
	if err != nil {
		return err
	}

	jsonBytes, err := lingo.MarshalLValue(value)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(d.dialect.Rebind(`
		INSERT INTO player_attributes (app_id, user_id, attr_name, value_json)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(app_id, user_id, attr_name) DO UPDATE SET value_json=excluded.value_json`),
		appID, userID, attrName, string(jsonBytes))
	return err
}

func (d *sqlDB) GetPlayerAttribute(appName, userID, attrName string) (lingo.LValue, error) {
	appID, err := d.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return d.scanAttribute(
		d.dialect.Rebind("SELECT value_json FROM player_attributes WHERE app_id = ? AND user_id = ? AND attr_name = ?"),
		appID, userID, attrName)
}

func (d *sqlDB) GetPlayerAttributeNames(appName, userID string) ([]string, error) {
	appID, err := d.getAppID(appName)
	if err != nil {
		return nil, err
	}
	return d.queryNames(d.dialect.Rebind("SELECT attr_name FROM player_attributes WHERE app_id = ? AND user_id = ?"), appID, userID)
}

func (d *sqlDB) DeletePlayerAttribute(appName, userID, attrName string) error {
	appID, err := d.getAppID(appName)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(d.dialect.Rebind("DELETE FROM player_attributes WHERE app_id = ? AND user_id = ? AND attr_name = ?"), appID, userID, attrName)
	return err
}

// --- helpers ---

func (d *sqlDB) queryNames(query string, args ...interface{}) ([]string, error) {
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (d *sqlDB) scanAttribute(query string, args ...interface{}) (lingo.LValue, error) {
	var valueJSON string

	err := d.db.QueryRow(query, args...).Scan(&valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return lingo.NewLVoid(), nil
		}
		return lingo.NewLVoid(), err
	}

	return lingo.UnmarshalLValue([]byte(valueJSON))
}
