package outbound

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"fsos-server/internal/domain/ports"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresDB implements ports.DBAdapter and ports.MigrationTracker against a
// PostgreSQL server. It mirrors SQLiteDB but speaks Postgres SQL: $N
// placeholders, BIGSERIAL identities, TIMESTAMPTZ, and now() defaults.
type PostgresDB struct {
	sqlDB
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

	p := &PostgresDB{sqlDB{db: db, dialect: postgresDialect{}}}
	if err := p.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return p, nil
}

func (p *PostgresDB) init() error {
	return p.ensureMigrationsTable()
}

// --- DBUser ---

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

func (p *PostgresDB) AddColumn(table string, col ports.Column) error {
	if err := validateIdentifier(table); err != nil {
		return fmt.Errorf("invalid table name %q: %w", table, err)
	}
	if err := validateIdentifier(col.Name); err != nil {
		return fmt.Errorf("invalid column name %q: %w", col.Name, err)
	}
	var b strings.Builder
	b.WriteString("ALTER TABLE ")
	b.WriteString(table)
	b.WriteString(" ADD COLUMN IF NOT EXISTS ")
	b.WriteString(col.Name)
	b.WriteString(" ")
	b.WriteString(p.columnTypeSQL(col.Type))
	if col.IsNotNull {
		b.WriteString(" NOT NULL")
	}
	if col.IsUnique {
		b.WriteString(" UNIQUE")
	}
	if col.DefType != ports.DefaultNone {
		b.WriteString(" DEFAULT ")
		b.WriteString(p.defaultSQL(col))
	}
	_, err := p.db.Exec(b.String())
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
