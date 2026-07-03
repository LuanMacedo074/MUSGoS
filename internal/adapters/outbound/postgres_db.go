package outbound

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresDB implements ports.DBAdapter and ports.MigrationTracker against a
// PostgreSQL server. It mirrors SQLiteDB but speaks Postgres SQL: $N
// placeholders, BIGSERIAL identities, TIMESTAMPTZ, and now() defaults.
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB opens a connection pool for the given DSN. The DSN may be a
// URL ("postgres://user:pass@host:5432/db?sslmode=disable") or a keyword string
// ("host=localhost user=... dbname=..."). It pings the server to fail fast.
func NewPostgresDB(dsn string) (*PostgresDB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("postgres: empty connection string (set DATABASE_URL)")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	p := &PostgresDB{db: db}
	if err := p.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return p, nil
}

func (p *PostgresDB) init() error {
	_, err := p.db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ NOT NULL
		)
	`)
	return err
}

// --- MigrationTracker ---

func (p *PostgresDB) GetAppliedMigrations() ([]string, error) {
	rows, err := p.db.Query("SELECT name FROM migrations ORDER BY name")
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

func (p *PostgresDB) MarkMigrationApplied(name string) error {
	_, err := p.db.Exec("INSERT INTO migrations (name, applied_at) VALUES ($1, $2)", name, time.Now())
	return err
}

// --- DBAdmin ---

func (p *PostgresDB) CreateApplication(appName string) error {
	_, err := p.db.Exec("INSERT INTO applications (name) VALUES ($1)", appName)
	return err
}

func (p *PostgresDB) DeleteApplication(appName string) error {
	result, err := p.db.Exec("DELETE FROM applications WHERE name = $1", appName)
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

func (p *PostgresDB) SetApplicationAttribute(appName, attrName string, value lingo.LValue) error {
	appID, err := p.getAppID(appName)
	if err != nil {
		return err
	}

	jsonBytes, err := lingo.MarshalLValue(value)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`
		INSERT INTO application_attributes (app_id, attr_name, value_json)
		VALUES ($1, $2, $3)
		ON CONFLICT(app_id, attr_name) DO UPDATE SET value_json=excluded.value_json`,
		appID, attrName, string(jsonBytes))
	return err
}

func (p *PostgresDB) GetApplicationAttribute(appName, attrName string) (lingo.LValue, error) {
	appID, err := p.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return p.scanAttribute(
		"SELECT value_json FROM application_attributes WHERE app_id = $1 AND attr_name = $2",
		appID, attrName)
}

func (p *PostgresDB) GetApplicationAttributeNames(appName string) ([]string, error) {
	appID, err := p.getAppID(appName)
	if err != nil {
		return nil, err
	}
	return p.queryNames("SELECT attr_name FROM application_attributes WHERE app_id = $1", appID)
}

func (p *PostgresDB) DeleteApplicationAttribute(appName, attrName string) error {
	appID, err := p.getAppID(appName)
	if err != nil {
		return err
	}
	_, err = p.db.Exec("DELETE FROM application_attributes WHERE app_id = $1 AND attr_name = $2", appID, attrName)
	return err
}

// --- DBPlayer ---

func (p *PostgresDB) SetPlayerAttribute(appName, userID, attrName string, value lingo.LValue) error {
	appID, err := p.getAppID(appName)
	if err != nil {
		return err
	}

	jsonBytes, err := lingo.MarshalLValue(value)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`
		INSERT INTO player_attributes (app_id, user_id, attr_name, value_json)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(app_id, user_id, attr_name) DO UPDATE SET value_json=excluded.value_json`,
		appID, userID, attrName, string(jsonBytes))
	return err
}

func (p *PostgresDB) GetPlayerAttribute(appName, userID, attrName string) (lingo.LValue, error) {
	appID, err := p.getAppID(appName)
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return p.scanAttribute(
		"SELECT value_json FROM player_attributes WHERE app_id = $1 AND user_id = $2 AND attr_name = $3",
		appID, userID, attrName)
}

func (p *PostgresDB) GetPlayerAttributeNames(appName, userID string) ([]string, error) {
	appID, err := p.getAppID(appName)
	if err != nil {
		return nil, err
	}
	return p.queryNames("SELECT attr_name FROM player_attributes WHERE app_id = $1 AND user_id = $2", appID, userID)
}

func (p *PostgresDB) DeletePlayerAttribute(appName, userID, attrName string) error {
	appID, err := p.getAppID(appName)
	if err != nil {
		return err
	}
	_, err = p.db.Exec("DELETE FROM player_attributes WHERE app_id = $1 AND user_id = $2 AND attr_name = $3", appID, userID, attrName)
	return err
}

// --- DBUser ---

func (p *PostgresDB) CreateUser(username, passwordHash string, userLevel int) error {
	_, err := p.db.Exec(
		"INSERT INTO users (uuid, username, password_hash, user_level) VALUES ($1, $2, $3, $4)",
		uuid.New().String(), username, passwordHash, userLevel)
	return err
}

func (p *PostgresDB) GetUser(username string) (*ports.User, error) {
	var u ports.User
	err := p.db.QueryRow(
		"SELECT id, uuid, username, password_hash, user_level, created_at FROM users WHERE username = $1",
		username).Scan(&u.ID, &u.UUID, &u.Username, &u.PasswordHash, &u.UserLevel, &u.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ports.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (p *PostgresDB) DeleteUser(username string) error {
	result, err := p.db.Exec("DELETE FROM users WHERE username = $1", username)
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

func (p *PostgresDB) UpdateUserLevel(username string, level int) error {
	result, err := p.db.Exec("UPDATE users SET user_level = $1 WHERE username = $2", level, username)
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

func (p *PostgresDB) UpdateUserPassword(username, passwordHash string) error {
	result, err := p.db.Exec(
		"UPDATE users SET password_hash = $1 WHERE username = $2",
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

func (p *PostgresDB) CreateBan(userID *int64, ipAddress *string, reason string, expiresAt *time.Time) error {
	_, err := p.db.Exec(
		"INSERT INTO bans (uuid, user_id, ip_address, reason, expires_at) VALUES ($1, $2, $3, $4, $5)",
		uuid.New().String(), userID, ipAddress, reason, expiresAt)
	return err
}

func (p *PostgresDB) GetActiveBanByUserID(userID int64) (*ports.Ban, error) {
	var b ports.Ban
	err := p.db.QueryRow(`
		SELECT id, uuid, user_id, ip_address, reason, expires_at, revoked_at, created_at
		FROM bans
		WHERE user_id = $1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now())
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

func (p *PostgresDB) GetActiveBanByIP(ipAddress string) (*ports.Ban, error) {
	var b ports.Ban
	err := p.db.QueryRow(`
		SELECT id, uuid, user_id, ip_address, reason, expires_at, revoked_at, created_at
		FROM bans
		WHERE ip_address = $1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now())
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

func (p *PostgresDB) RevokeBan(banID int64) error {
	result, err := p.db.Exec(
		"UPDATE bans SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL",
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

func (p *PostgresDB) CreateTable(def ports.Table) error {
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

	composite := p.hasCompositePK(def)

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

		// A single-column auto-increment primary key maps to BIGSERIAL, which
		// carries its own implicit type and NOT NULL — so skip the base type.
		autoPK := col.IsPK && col.IsAutoIncr && !composite
		if autoPK {
			b.WriteString("BIGSERIAL")
		} else {
			b.WriteString(p.columnTypeSQL(col.Type))
		}

		if col.IsPK && !composite {
			b.WriteString(" PRIMARY KEY")
		}
		if col.IsNotNull && !autoPK {
			b.WriteString(" NOT NULL")
		}
		if col.IsUnique {
			b.WriteString(" UNIQUE")
		}
		if col.DefType != ports.DefaultNone {
			b.WriteString(" DEFAULT ")
			b.WriteString(p.defaultSQL(col))
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

	_, err := p.db.Exec(b.String())
	return err
}

func (p *PostgresDB) DropTable(name string) error {
	if err := validateIdentifier(name); err != nil {
		return fmt.Errorf("invalid table name %q: %w", name, err)
	}
	_, err := p.db.Exec("DROP TABLE IF EXISTS " + name + " CASCADE")
	return err
}

func (p *PostgresDB) CreateIndex(def ports.Index) error {
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
	_, err := p.db.Exec(stmt)
	return err
}

func (p *PostgresDB) columnTypeSQL(t ports.ColumnType) string {
	switch t {
	case ports.ColInteger:
		return "BIGINT"
	case ports.ColText:
		return "TEXT"
	case ports.ColDatetime:
		return "TIMESTAMPTZ"
	default:
		return "TEXT"
	}
}

func (p *PostgresDB) defaultSQL(col ports.Column) string {
	switch col.DefType {
	case ports.DefaultNow:
		return "now()"
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

func (p *PostgresDB) hasCompositePK(def ports.Table) bool {
	return len(def.PrimaryKeys) > 0
}

// QueryBuilder returns a generic query builder for this database.
func (p *PostgresDB) QueryBuilder() ports.QueryBuilder {
	return NewPostgresQueryBuilder(p.db)
}

// --- Close ---

func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// --- helpers ---

func (p *PostgresDB) getAppID(appName string) (int64, error) {
	var id int64
	err := p.db.QueryRow("SELECT id FROM applications WHERE name = $1", appName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("application %q not found: %w", appName, err)
	}
	return id, nil
}

func (p *PostgresDB) queryNames(query string, args ...interface{}) ([]string, error) {
	rows, err := p.db.Query(query, args...)
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

func (p *PostgresDB) scanAttribute(query string, args ...interface{}) (lingo.LValue, error) {
	var valueJSON string

	err := p.db.QueryRow(query, args...).Scan(&valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return lingo.NewLVoid(), nil
		}
		return lingo.NewLVoid(), err
	}

	return lingo.UnmarshalLValue([]byte(valueJSON))
}
