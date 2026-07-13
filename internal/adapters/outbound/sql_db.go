package outbound

import (
	"database/sql"
	"fmt"
	"time"

	"fsos-server/internal/domain/ports"
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
