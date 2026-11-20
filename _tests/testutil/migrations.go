package testutil

import (
	"errors"
	"testing"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
)

// MigrationDB is the exported storage boundary used by migration scenarios.
type MigrationDB interface {
	ports.DBAdapter
	ports.MigrationTracker
	QueryBuilder() ports.QueryBuilder
}

// TableExists reports whether the adapter can query the named table.
func TableExists(db MigrationDB, table string) bool {
	_, err := db.QueryBuilder().Table(table).Count()
	return err == nil
}

// RunMigrations applies the supplied migrations through the public runner.
func RunMigrations(db MigrationDB, ms ...ports.Migration) error {
	_, err := services.NewMigrationRunner(db, db, ms).RunPending()
	return err
}

// IsApplied reports whether the migration ledger contains name.
func IsApplied(t testing.TB, db MigrationDB, name string) bool {
	t.Helper()
	applied, err := db.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range applied {
		if a == name {
			return true
		}
	}
	return false
}

// CountRows reads a fixture table through the public query builder.
func CountRows(t testing.TB, db MigrationDB, table string) int64 {
	t.Helper()
	n, err := db.QueryBuilder().Table(table).Count()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return n
}

// SQLExecutor obtains the raw SQL capability of a migration transaction.
func SQLExecutor(db ports.DBAdapter) (ports.SQLMigrationExecutor, error) {
	executor, ok := db.(ports.SQLMigrationExecutor)
	if !ok {
		return nil, errors.New("migration adapter cannot execute SQL migrations")
	}
	return executor, nil
}
