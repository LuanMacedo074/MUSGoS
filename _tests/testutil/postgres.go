package testutil

import (
	"os"
	"testing"

	"fsos-server/internal/adapters/outbound"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// NewPostgresFixture opens TEST_POSTGRES_DSN and clears its tables and migration
// ledger. Callers reset their own non-table objects before installing migrations.
// The configured database is disposable; packages sharing it must run sequentially.
func NewPostgresFixture(t testing.TB) *outbound.PostgresDB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping postgres integration tests")
	}

	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("failed to parse test postgres config: %v", err)
	}
	// Migrations change column layouts between reads, so cached SELECT * plans
	// cannot be reused during setup. Keep this setting local to the test pool.
	config.DefaultQueryExecMode = pgx.QueryExecModeExec
	connString := stdlib.RegisterConnConfig(config)
	t.Cleanup(func() { stdlib.UnregisterConnConfig(connString) })
	db, err := outbound.NewPostgresDB(connString)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	resetPostgresSchema(t, db)

	return db
}

func resetPostgresSchema(t testing.TB, db *outbound.PostgresDB) {
	t.Helper()
	// Remove tables created by previous adapter scenarios before replaying the schema.
	tables, err := db.QueryBuilder().Table("pg_tables").Where("schemaname", "public").Get()
	if err != nil {
		t.Fatalf("failed to list test schema tables: %v", err)
	}
	for _, row := range tables {
		tbl := row["tablename"].(string)
		if tbl == "migrations" {
			continue // the adapter bootstraps this table before reset
		}
		if err := db.DropTable(tbl); err != nil {
			t.Fatalf("failed to drop %s: %v", tbl, err)
		}
	}
	if _, err := db.QueryBuilder().Table("migrations").Delete(); err != nil {
		t.Fatalf("failed to clear migrations ledger: %v", err)
	}
}
