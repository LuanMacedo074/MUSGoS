package outbound

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	_ "modernc.org/sqlite"
)

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

func (s *SQLiteDB) CreateUser(username, passwordHash, salt string, userLevel int) error {
	_, err := s.db.Exec(
		"INSERT INTO users (username, password_hash, salt, user_level) VALUES (?, ?, ?, ?)",
		username, passwordHash, salt, userLevel)
	return err
}

func (s *SQLiteDB) GetUser(username string) (*ports.User, error) {
	var u ports.User
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, salt, user_level, created_at FROM users WHERE username = ?",
		username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Salt, &u.UserLevel, &u.CreatedAt)
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

func (s *SQLiteDB) UpdateUserPassword(username, passwordHash, salt string) error {
	result, err := s.db.Exec(
		"UPDATE users SET password_hash = ?, salt = ? WHERE username = ?",
		passwordHash, salt, username)
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
		"INSERT INTO bans (user_id, ip_address, reason, expires_at) VALUES (?, ?, ?, ?)",
		userID, ipAddress, reason, expiresAt)
	return err
}

func (s *SQLiteDB) GetActiveBanByUserID(userID int64) (*ports.Ban, error) {
	var b ports.Ban
	err := s.db.QueryRow(`
		SELECT id, user_id, ip_address, reason, expires_at, revoked_at, created_at
		FROM bans
		WHERE user_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC LIMIT 1`,
		userID).Scan(&b.ID, &b.UserID, &b.IPAddress, &b.Reason, &b.ExpiresAt, &b.RevokedAt, &b.CreatedAt)
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
		SELECT id, user_id, ip_address, reason, expires_at, revoked_at, created_at
		FROM bans
		WHERE ip_address = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC LIMIT 1`,
		ipAddress).Scan(&b.ID, &b.UserID, &b.IPAddress, &b.Reason, &b.ExpiresAt, &b.RevokedAt, &b.CreatedAt)
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

// --- ExecRaw ---

func (s *SQLiteDB) ExecRaw(sqlStr string) error {
	_, err := s.db.Exec(sqlStr)
	return err
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
