//go:build integration

package factory_test

import (
	"os"
	"testing"

	"fsos-server/internal/factory"
)

// Runs against the Postgres brought up by docker/thirdparties. The DSN defaults
// to that stack; scripts/run-tests.sh sets TEST_POSTGRES_DSN from
// custom_settings.env.
func TestNewDatabase_PostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:my_secret_pw@127.0.0.1:5432/musgo_regression?sslmode=disable"
	}

	result, err := factory.NewDatabase("postgres", dsn, nil)
	if err != nil {
		t.Skipf("postgres not reachable (run 'make thirdparties-up'): %v", err)
	}
	defer result.Adapter.Close()

	if result.Adapter == nil || result.MigrationRunner == nil || result.QueryBuilder == nil {
		t.Error("database result should be fully populated")
	}
}
