package outbound

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	"github.com/google/uuid"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

var validOnDelete = map[string]bool{
	"":            true,
	"CASCADE":     true,
	"SET NULL":    true,
	"SET DEFAULT": true,
	"RESTRICT":    true,
	"NO ACTION":   true,
}

func validateIdentifier(name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("identifier must match [a-zA-Z_][a-zA-Z0-9_]*")
	}
	return nil
}

// sqlDB is the storage core shared by the SQL-backed adapters (RFC-007). It
// owns the connection pool and defers backend differences to its dialect; the
// duplicated method bodies in the sqlite/postgres adapters migrate here group
// by group, written once with ?-placeholders and rebound per dialect.
type sqlDB struct {
	db      *sql.DB
	dialect dialect
}

// init runs per-connection setup for the backend and bootstraps the
// migrations table; called from the adapters' constructors.
func (d *sqlDB) init() error {
	if err := d.dialect.Init(d.db); err != nil {
		return err
	}
	return d.ensureMigrationsTable()
}

func (d *sqlDB) Close() error {
	return d.db.Close()
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

// --- DBUser ---

func (d *sqlDB) CreateUser(username, passwordHash string, userLevel int) error {
	_, err := d.db.Exec(
		d.dialect.Rebind("INSERT INTO users (uuid, username, password_hash, user_level) VALUES (?, ?, ?, ?)"),
		uuid.New().String(), username, passwordHash, userLevel)
	return err
}

func (d *sqlDB) GetUser(username string) (*ports.User, error) {
	var u ports.User
	err := d.db.QueryRow(
		d.dialect.Rebind("SELECT id, uuid, username, password_hash, user_level, created_at FROM users WHERE username = ?"),
		username).Scan(&u.ID, &u.UUID, &u.Username, &u.PasswordHash, &u.UserLevel, &u.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ports.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (d *sqlDB) DeleteUser(username string) error {
	result, err := d.db.Exec(d.dialect.Rebind("DELETE FROM users WHERE username = ?"), username)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ports.ErrUserNotFound
	}
	return nil
}

func (d *sqlDB) UpdateUserLevel(username string, level int) error {
	result, err := d.db.Exec(d.dialect.Rebind("UPDATE users SET user_level = ? WHERE username = ?"), level, username)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ports.ErrUserNotFound
	}
	return nil
}

func (d *sqlDB) UpdateUserPassword(username, passwordHash string) error {
	result, err := d.db.Exec(
		d.dialect.Rebind("UPDATE users SET password_hash = ? WHERE username = ?"),
		passwordHash, username)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ports.ErrUserNotFound
	}
	return nil
}

// --- DBBan ---

func (d *sqlDB) CreateBan(userID *int64, ipAddress *string, reason string, expiresAt *time.Time) error {
	_, err := d.db.Exec(
		d.dialect.Rebind("INSERT INTO bans (uuid, user_id, ip_address, reason, expires_at) VALUES (?, ?, ?, ?, ?)"),
		uuid.New().String(), userID, ipAddress, reason, expiresAt)
	return err
}

func (d *sqlDB) GetActiveBanByUserID(userID int64) (*ports.Ban, error) {
	return d.getActiveBan("user_id", userID)
}

func (d *sqlDB) GetActiveBanByIP(ipAddress string) (*ports.Ban, error) {
	return d.getActiveBan("ip_address", ipAddress)
}

// getActiveBan looks up the newest unrevoked, unexpired ban by the given
// column ("user_id" or "ip_address" — fixed strings, never caller input).
func (d *sqlDB) getActiveBan(column string, value interface{}) (*ports.Ban, error) {
	var b ports.Ban
	query := fmt.Sprintf(`
		SELECT id, uuid, user_id, ip_address, reason, expires_at, revoked_at, created_at
		FROM bans
		WHERE %s = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > %s)
		ORDER BY created_at DESC LIMIT 1`, column, d.dialect.NowExpr())
	err := d.db.QueryRow(d.dialect.Rebind(query),
		value).Scan(&b.ID, &b.UUID, &b.UserID, &b.IPAddress, &b.Reason, &b.ExpiresAt, &b.RevokedAt, &b.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ports.ErrBanNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (d *sqlDB) RevokeBan(banID int64) error {
	result, err := d.db.Exec(
		d.dialect.Rebind(fmt.Sprintf("UPDATE bans SET revoked_at = %s WHERE id = ? AND revoked_at IS NULL", d.dialect.NowExpr())),
		banID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ports.ErrBanNotFound
	}
	return nil
}

// --- Schema operations ---

func (d *sqlDB) CreateTable(def ports.Table) error {
	if err := validateIdentifier(def.Name); err != nil {
		return fmt.Errorf("invalid table name %q: %w", def.Name, err)
	}
	for _, col := range def.Columns {
		if err := validateIdentifier(col.Name); err != nil {
			return fmt.Errorf("invalid column name %q: %w", col.Name, err)
		}
	}
	for _, fk := range def.ForeignKeys {
		for _, id := range []string{fk.Column, fk.RefTable, fk.RefCol} {
			if err := validateIdentifier(id); err != nil {
				return fmt.Errorf("invalid identifier %q in foreign key: %w", id, err)
			}
		}
		if !validOnDelete[fk.OnDelete] {
			return fmt.Errorf("invalid ON DELETE action %q", fk.OnDelete)
		}
	}
	for _, pk := range def.PrimaryKeys {
		if err := validateIdentifier(pk); err != nil {
			return fmt.Errorf("invalid primary key column %q: %w", pk, err)
		}
	}
	for _, col := range def.RequireOneOf {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid column %q in require_one_of: %w", col, err)
		}
	}

	composite := len(def.PrimaryKeys) > 0

	var b strings.Builder
	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(def.Name)
	b.WriteString(" (\n")

	for i, col := range def.Columns {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("\t")
		b.WriteString(col.Name)
		b.WriteString(" ")

		// A single-column auto-increment primary key is one dialect fragment
		// (INTEGER PRIMARY KEY AUTOINCREMENT / BIGSERIAL PRIMARY KEY); both
		// spellings imply NOT NULL, so the constraint chain is skipped.
		autoPK := col.IsPK && col.IsAutoIncr && !composite
		if autoPK {
			b.WriteString(d.dialect.AutoIncrPKSQL())
		} else {
			b.WriteString(d.dialect.ColumnType(col.Type))
			if col.IsPK && !composite {
				b.WriteString(" PRIMARY KEY")
			}
			if col.IsNotNull {
				b.WriteString(" NOT NULL")
			}
		}
		if col.IsUnique {
			b.WriteString(" UNIQUE")
		}
		if col.DefType != ports.DefaultNone {
			b.WriteString(" DEFAULT ")
			b.WriteString(d.defaultSQL(col))
		}
	}

	if composite {
		b.WriteString(",\n\tPRIMARY KEY (")
		b.WriteString(strings.Join(def.PrimaryKeys, ", "))
		b.WriteString(")")
	}

	for _, fk := range def.ForeignKeys {
		b.WriteString(",\n\tFOREIGN KEY (")
		b.WriteString(fk.Column)
		b.WriteString(") REFERENCES ")
		b.WriteString(fk.RefTable)
		b.WriteString("(")
		b.WriteString(fk.RefCol)
		b.WriteString(")")
		if fk.OnDelete != "" {
			b.WriteString(" ON DELETE ")
			b.WriteString(fk.OnDelete)
		}
	}

	if len(def.RequireOneOf) > 0 {
		b.WriteString(",\n\tCHECK (")
		for i, col := range def.RequireOneOf {
			if i > 0 {
				b.WriteString(" OR ")
			}
			b.WriteString(col)
			b.WriteString(" IS NOT NULL")
		}
		b.WriteString(")")
	}

	b.WriteString("\n)")

	_, err := d.db.Exec(b.String())
	return err
}

func (d *sqlDB) DropTable(name string) error {
	if err := validateIdentifier(name); err != nil {
		return fmt.Errorf("invalid table name %q: %w", name, err)
	}
	_, err := d.db.Exec("DROP TABLE IF EXISTS " + name + d.dialect.DropTableSuffix())
	return err
}

func (d *sqlDB) AddColumn(table string, col ports.Column) error {
	if err := validateIdentifier(table); err != nil {
		return fmt.Errorf("invalid table name %q: %w", table, err)
	}
	if err := validateIdentifier(col.Name); err != nil {
		return fmt.Errorf("invalid column name %q: %w", col.Name, err)
	}
	if skip, err := d.dialect.SkipAddColumn(d.db, table, col.Name); err != nil || skip {
		return err
	}

	var b strings.Builder
	b.WriteString("ALTER TABLE ")
	b.WriteString(table)
	b.WriteString(" ")
	b.WriteString(d.dialect.AddColumnClause())
	b.WriteString(col.Name)
	b.WriteString(" ")
	b.WriteString(d.dialect.ColumnType(col.Type))
	if col.IsNotNull {
		b.WriteString(" NOT NULL")
	}
	if col.IsUnique && d.dialect.SupportsUniqueInAddColumn() {
		b.WriteString(" UNIQUE")
	}
	if col.DefType != ports.DefaultNone {
		b.WriteString(" DEFAULT ")
		b.WriteString(d.defaultSQL(col))
	}
	_, err := d.db.Exec(b.String())
	return err
}

func (d *sqlDB) CreateIndex(def ports.Index) error {
	if err := validateIdentifier(def.Name); err != nil {
		return fmt.Errorf("invalid index name %q: %w", def.Name, err)
	}
	if err := validateIdentifier(def.Table); err != nil {
		return fmt.Errorf("invalid table name %q: %w", def.Table, err)
	}
	for _, col := range def.Columns {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid column name %q: %w", col, err)
		}
	}
	stmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
		def.Name, def.Table, strings.Join(def.Columns, ", "))
	_, err := d.db.Exec(stmt)
	return err
}

func (d *sqlDB) defaultSQL(col ports.Column) string {
	switch col.DefType {
	case ports.DefaultNow:
		return d.dialect.DefaultNowExpr()
	case ports.DefaultLiteral:
		switch v := col.DefaultVal.(type) {
		case string:
			escaped := strings.ReplaceAll(v, "'", "''")
			return fmt.Sprintf("'%s'", escaped)
		default:
			return fmt.Sprintf("%v", v)
		}
	default:
		return ""
	}
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
