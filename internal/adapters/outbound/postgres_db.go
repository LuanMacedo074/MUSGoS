package outbound

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"fsos-server/internal/domain/ports"

	"github.com/jackc/pgx/v5/stdlib"
)

// PostgresDB implements ports.DBAdapter and ports.MigrationTracker against a
// PostgreSQL server. All persistence logic lives on the embedded storage
// core; this type only owns construction (DSN validation, fail-fast ping).
type PostgresDB struct {
	sqlDB
}

// The migration runner only applies migrations atomically on adapters that
// implement ports.MigrationTransactor; fail the build if this one stops.
var _ ports.MigrationTransactor = (*PostgresDB)(nil)

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

	p := &PostgresDB{newSQLDB(db, postgresDialect{})}
	if err := p.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return p, nil
}

// BeginMigration clears pgx's statement and description caches on the exact
// connection that will run the next migration. Earlier migrations may have
// changed the result shape of cached SELECTs. Clearing only server-side plans
// would leave pgx's client-side caches stale.
func (p *PostgresDB) BeginMigration() (ports.MigrationTx, error) {
	ctx := context.Background()
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if err := conn.Raw(func(driverConn interface{}) error {
		return driverConn.(*stdlib.Conn).Conn().DeallocateAll(ctx)
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("clear migration statement cache: %w", err)
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &postgresMigrationTx{
		sqlMigrationTx: &sqlMigrationTx{sqlDB: sqlDB{db: p.db, exec: tx, dialect: p.dialect}, tx: tx},
		conn:           conn,
	}, nil
}

// Keep the reserved connection until the migration transaction is finalized.
type postgresMigrationTx struct {
	*sqlMigrationTx
	conn *sql.Conn
}

func (t *postgresMigrationTx) Commit() error {
	defer t.conn.Close()
	return t.sqlMigrationTx.Commit()
}

func (t *postgresMigrationTx) Rollback() error {
	defer t.conn.Close()
	return t.sqlMigrationTx.Rollback()
}
