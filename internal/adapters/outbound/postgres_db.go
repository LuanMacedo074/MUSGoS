package outbound

import (
	"database/sql"
	"fmt"
	"strings"

	"fsos-server/internal/domain/ports"

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

// QueryBuilder returns a generic query builder for this database.
func (p *PostgresDB) QueryBuilder() ports.QueryBuilder {
	return NewPostgresQueryBuilder(p.db)
}

// --- Close ---

func (p *PostgresDB) Close() error {
	return p.db.Close()
}
