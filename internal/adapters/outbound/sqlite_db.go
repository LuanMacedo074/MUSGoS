package outbound

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

var validOnDelete = map[string]bool{
	"":           true,
	"CASCADE":    true,
	"SET NULL":   true,
	"SET DEFAULT": true,
	"RESTRICT":   true,
	"NO ACTION":  true,
}

func validateIdentifier(name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("identifier must match [a-zA-Z_][a-zA-Z0-9_]*")
	}
	return nil
}

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	s := &SQLiteDB{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return s, nil
}

func (s *SQLiteDB) init() error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
	}
	for _, p := range pragmas {
		if _, err := s.db.Exec(p); err != nil {
			return fmt.Errorf("failed to set %s: %w", p, err)
		}
	}

	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at DATETIME NOT NULL
		)
	`)
	return err
}

// --- MigrationTracker ---

func (s *SQLiteDB) GetAppliedMigrations() ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM migrations ORDER BY name")
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

func (s *SQLiteDB) MarkMigrationApplied(name string) error {
	_, err := s.db.Exec("INSERT INTO migrations (name, applied_at) VALUES (?, ?)", name, time.Now())
	return err
}

// --- DBAdmin ---

func (s *SQLiteDB) CreateApplication(appName string) error {
	_, err := s.db.Exec("INSERT INTO applications (name) VALUES (?)", appName)
	return err
}

func (s *SQLiteDB) DeleteApplication(appName string) error {
	result, err := s.db.Exec("DELETE FROM applications WHERE name = ?", appName)
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

// --- DBApplication ---

func (s *SQLiteDB) SetApplicationAttribute(appName, attrName string, value lingo.LValue) error {
	appID, err := s.getAppID(appName)
	if err != nil {
		return err
	}

	jsonBytes, err := lingo.MarshalLValue(value)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO application_attributes (app_id, attr_name, value_json)
		VALUES (?, ?, ?)
		ON CONFLICT(app_id, attr_name) DO UPDATE SET value_json=excluded.value_json`,
		appID, attrName, string(jsonBytes))
	return err
}

func (s *SQLiteDB) GetApplicationAttribute(appName, attrName string) (lingo.LValue, error) {
	appID, err := s.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return s.scanAttribute(
		"SELECT value_json FROM application_attributes WHERE app_id = ? AND attr_name = ?",
		appID, attrName)
}

func (s *SQLiteDB) GetApplicationAttributeNames(appName string) ([]string, error) {
	appID, err := s.getAppID(appName)
	if err != nil {
		return nil, err
	}
	return s.queryNames("SELECT attr_name FROM application_attributes WHERE app_id = ?", appID)
}

func (s *SQLiteDB) DeleteApplicationAttribute(appName, attrName string) error {
	appID, err := s.getAppID(appName)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM application_attributes WHERE app_id = ? AND attr_name = ?", appID, attrName)
	return err
}

// --- DBPlayer ---

func (s *SQLiteDB) SetPlayerAttribute(appName, userID, attrName string, value lingo.LValue) error {
	appID, err := s.getAppID(appName)
	if err != nil {
		return err
	}

	jsonBytes, err := lingo.MarshalLValue(value)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO player_attributes (app_id, user_id, attr_name, value_json)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(app_id, user_id, attr_name) DO UPDATE SET value_json=excluded.value_json`,
		appID, userID, attrName, string(jsonBytes))
	return err
}

func (s *SQLiteDB) GetPlayerAttribute(appName, userID, attrName string) (lingo.LValue, error) {
	appID, err := s.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return s.scanAttribute(
		"SELECT value_json FROM player_attributes WHERE app_id = ? AND user_id = ? AND attr_name = ?",
		appID, userID, attrName)
}

func (s *SQLiteDB) GetPlayerAttributeNames(appName, userID string) ([]string, error) {
	appID, err := s.getAppID(appName)
	if err != nil {
		return nil, err
	}
	return s.queryNames("SELECT attr_name FROM player_attributes WHERE app_id = ? AND user_id = ?", appID, userID)
}

func (s *SQLiteDB) DeletePlayerAttribute(appName, userID, attrName string) error {
	appID, err := s.getAppID(appName)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM player_attributes WHERE app_id = ? AND user_id = ? AND attr_name = ?", appID, userID, attrName)
	return err
}

// --- DBUser ---

func (s *SQLiteDB) CreateUser(username, passwordHash string, userLevel int) error {
	_, err := s.db.Exec(
		"INSERT INTO users (uuid, username, password_hash, user_level) VALUES (?, ?, ?, ?)",
		uuid.New().String(), username, passwordHash, userLevel)
	return err
}

func (s *SQLiteDB) GetUser(username string) (*ports.User, error) {
	var u ports.User
	err := s.db.QueryRow(
		"SELECT id, uuid, username, password_hash, user_level, created_at FROM users WHERE username = ?",
		username).Scan(&u.ID, &u.UUID, &u.Username, &u.PasswordHash, &u.UserLevel, &u.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ports.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *SQLiteDB) DeleteUser(username string) error {
	result, err := s.db.Exec("DELETE FROM users WHERE username = ?", username)
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

func (s *SQLiteDB) UpdateUserLevel(username string, level int) error {
	result, err := s.db.Exec("UPDATE users SET user_level = ? WHERE username = ?", level, username)
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

func (s *SQLiteDB) UpdateUserPassword(username, passwordHash string) error {
	result, err := s.db.Exec(
		"UPDATE users SET password_hash = ? WHERE username = ?",
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

func (s *SQLiteDB) CreateBan(userID *int64, ipAddress *string, reason string, expiresAt *time.Time) error {
	_, err := s.db.Exec(
		"INSERT INTO bans (uuid, user_id, ip_address, reason, expires_at) VALUES (?, ?, ?, ?, ?)",
		uuid.New().String(), userID, ipAddress, reason, expiresAt)
	return err
}

func (s *SQLiteDB) GetActiveBanByUserID(userID int64) (*ports.Ban, error) {
	var b ports.Ban
	err := s.db.QueryRow(`
		SELECT id, uuid, user_id, ip_address, reason, expires_at, revoked_at, created_at
		FROM bans
		WHERE user_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC LIMIT 1`,
		userID).Scan(&b.ID, &b.UUID, &b.UserID, &b.IPAddress, &b.Reason, &b.ExpiresAt, &b.RevokedAt, &b.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ports.ErrBanNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (s *SQLiteDB) GetActiveBanByIP(ipAddress string) (*ports.Ban, error) {
	var b ports.Ban
	err := s.db.QueryRow(`
		SELECT id, uuid, user_id, ip_address, reason, expires_at, revoked_at, created_at
		FROM bans
		WHERE ip_address = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC LIMIT 1`,
		ipAddress).Scan(&b.ID, &b.UUID, &b.UserID, &b.IPAddress, &b.Reason, &b.ExpiresAt, &b.RevokedAt, &b.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ports.ErrBanNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (s *SQLiteDB) RevokeBan(banID int64) error {
	result, err := s.db.Exec(
		"UPDATE bans SET revoked_at = datetime('now') WHERE id = ? AND revoked_at IS NULL",
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

func (s *SQLiteDB) CreateTable(def ports.Table) error {
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
		b.WriteString(s.columnTypeSQL(col.Type))

		if col.IsPK && !s.hasCompositePK(def) {
			b.WriteString(" PRIMARY KEY")
			if col.IsAutoIncr {
				b.WriteString(" AUTOINCREMENT")
			}
		}
		if col.IsNotNull {
			b.WriteString(" NOT NULL")
		}
		if col.IsUnique {
			b.WriteString(" UNIQUE")
		}
		if col.DefType != ports.DefaultNone {
			b.WriteString(" DEFAULT ")
			b.WriteString(s.defaultSQL(col))
		}
	}

	if s.hasCompositePK(def) {
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

	_, err := s.db.Exec(b.String())
	return err
}

func (s *SQLiteDB) DropTable(name string) error {
	if err := validateIdentifier(name); err != nil {
		return fmt.Errorf("invalid table name %q: %w", name, err)
	}
	_, err := s.db.Exec("DROP TABLE IF EXISTS " + name)
	return err
}

func (s *SQLiteDB) CreateIndex(def ports.Index) error {
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
	sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
		def.Name, def.Table, strings.Join(def.Columns, ", "))
	_, err := s.db.Exec(sql)
	return err
}

func (s *SQLiteDB) columnTypeSQL(t ports.ColumnType) string {
	switch t {
	case ports.ColInteger:
		return "INTEGER"
	case ports.ColText:
		return "TEXT"
	case ports.ColDatetime:
		return "DATETIME"
	default:
		return "TEXT"
	}
}

func (s *SQLiteDB) defaultSQL(col ports.Column) string {
	switch col.DefType {
	case ports.DefaultNow:
		return "(datetime('now'))"
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

func (s *SQLiteDB) hasCompositePK(def ports.Table) bool {
	return len(def.PrimaryKeys) > 0
}


// --- Close ---

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// --- helpers ---

func (s *SQLiteDB) getAppID(appName string) (int64, error) {
	var id int64
	err := s.db.QueryRow("SELECT id FROM applications WHERE name = ?", appName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("application %q not found: %w", appName, err)
	}
	return id, nil
}

func (s *SQLiteDB) queryNames(query string, args ...interface{}) ([]string, error) {
	rows, err := s.db.Query(query, args...)
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

func (s *SQLiteDB) scanAttribute(query string, args ...interface{}) (lingo.LValue, error) {
	var valueJSON string

	err := s.db.QueryRow(query, args...).Scan(&valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return lingo.NewLVoid(), nil
		}
		return lingo.NewLVoid(), err
	}

	return lingo.UnmarshalLValue([]byte(valueJSON))
}
