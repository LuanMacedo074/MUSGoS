package outbound

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	vType, vInt, vReal, vText, vBlob := marshalLValue(value)

	_, err = s.db.Exec(`
		INSERT INTO application_attributes (app_id, attr_name, value_type, value_int, value_real, value_text, value_blob)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(app_id, attr_name) DO UPDATE SET
			value_type=excluded.value_type, value_int=excluded.value_int,
			value_real=excluded.value_real, value_text=excluded.value_text,
			value_blob=excluded.value_blob`,
		appID, attrName, vType, vInt, vReal, vText, vBlob)
	return err
}

func (s *SQLiteDB) GetApplicationAttribute(appName, attrName string) (lingo.LValue, error) {
	appID, err := s.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return s.scanAttribute(
		"SELECT value_type, value_int, value_real, value_text, value_blob FROM application_attributes WHERE app_id = ? AND attr_name = ?",
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

	vType, vInt, vReal, vText, vBlob := marshalLValue(value)

	_, err = s.db.Exec(`
		INSERT INTO player_attributes (app_id, user_id, attr_name, value_type, value_int, value_real, value_text, value_blob)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(app_id, user_id, attr_name) DO UPDATE SET
			value_type=excluded.value_type, value_int=excluded.value_int,
			value_real=excluded.value_real, value_text=excluded.value_text,
			value_blob=excluded.value_blob`,
		appID, userID, attrName, vType, vInt, vReal, vText, vBlob)
	return err
}

func (s *SQLiteDB) GetPlayerAttribute(appName, userID, attrName string) (lingo.LValue, error) {
	appID, err := s.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return s.scanAttribute(
		"SELECT value_type, value_int, value_real, value_text, value_blob FROM player_attributes WHERE app_id = ? AND user_id = ? AND attr_name = ?",
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

func (s *SQLiteDB) SetUserAttribute(clientID, attrName string, value lingo.LValue) error {
	vType, vInt, vReal, vText, vBlob := marshalLValue(value)

	_, err := s.db.Exec(`
		INSERT INTO user_attributes (client_id, attr_name, value_type, value_int, value_real, value_text, value_blob)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id, attr_name) DO UPDATE SET
			value_type=excluded.value_type, value_int=excluded.value_int,
			value_real=excluded.value_real, value_text=excluded.value_text,
			value_blob=excluded.value_blob`,
		clientID, attrName, vType, vInt, vReal, vText, vBlob)
	return err
}

func (s *SQLiteDB) GetUserAttribute(clientID, attrName string) (lingo.LValue, error) {
	return s.scanAttribute(
		"SELECT value_type, value_int, value_real, value_text, value_blob FROM user_attributes WHERE client_id = ? AND attr_name = ?",
		clientID, attrName)
}

func (s *SQLiteDB) GetUserAttributeNames(clientID string) ([]string, error) {
	return s.queryNames("SELECT attr_name FROM user_attributes WHERE client_id = ?", clientID)
}

func (s *SQLiteDB) DeleteUserAttribute(clientID, attrName string) error {
	_, err := s.db.Exec("DELETE FROM user_attributes WHERE client_id = ? AND attr_name = ?", clientID, attrName)
	return err
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
	var (
		vType string
		vInt  sql.NullInt64
		vReal sql.NullFloat64
		vText sql.NullString
		vBlob []byte
	)

	err := s.db.QueryRow(query, args...).Scan(&vType, &vInt, &vReal, &vText, &vBlob)
	if err != nil {
		if err == sql.ErrNoRows {
			return lingo.NewLVoid(), nil
		}
		return lingo.NewLVoid(), err
	}

	return unmarshalLValue(vType, vInt, vReal, vText, vBlob), nil
}

func marshalLValue(value lingo.LValue) (vType string, vInt *int64, vReal *float64, vText *string, vBlob []byte) {
	switch v := value.(type) {
	case *lingo.LInteger:
		vType = "integer"
		i := int64(v.Value)
		vInt = &i
	case *lingo.LFloat:
		vType = "float"
		f := v.Value
		vReal = &f
	case *lingo.LString:
		vType = "string"
		vText = &v.Value
	case *lingo.LSymbol:
		vType = "symbol"
		vText = &v.Value
	case *lingo.LPropList:
		vType = "proplist"
		vBlob = v.GetBytes()
	case *lingo.LList:
		vType = "list"
		vBlob = v.GetBytes()
	default:
		vType = "void"
	}
	return
}

func unmarshalLValue(vType string, vInt sql.NullInt64, vReal sql.NullFloat64, vText sql.NullString, vBlob []byte) lingo.LValue {
	switch vType {
	case "integer":
		if vInt.Valid {
			return lingo.NewLInteger(int32(vInt.Int64))
		}
	case "float":
		if vReal.Valid {
			return lingo.NewLFloat(vReal.Float64)
		}
	case "string":
		if vText.Valid {
			return lingo.NewLString(vText.String)
		}
	case "symbol":
		if vText.Valid {
			return lingo.NewLSymbol(vText.String)
		}
	case "proplist", "list":
		if len(vBlob) > 0 {
			return lingo.FromRawBytes(vBlob, 0)
		}
	}
	return lingo.NewLVoid()
}
