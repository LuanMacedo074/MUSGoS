package factory_test

import (
	"path/filepath"
	"testing"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/factory"
)

func TestNewDatabase_SQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	result, err := factory.NewDatabase("sqlite", dbPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer result.Adapter.Close()

	if result.Adapter == nil {
		t.Error("adapter should not be nil")
	}
	if result.MigrationRunner == nil {
		t.Error("migration runner should not be nil")
	}
}

func TestNewDatabase_UnknownType(t *testing.T) {
	_, err := factory.NewDatabase("mongo", "/tmp/test.db", nil)
	if err == nil {
		t.Error("expected error for unsupported database type")
	}
}

func TestNewDatabase_RunsMigrations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ran := false
	m := &testMigration{
		name: "20260101000000_init",
		up:   func(db ports.DBAdapter) error { ran = true; return nil },
	}

	result, err := factory.NewDatabase("sqlite", dbPath, []ports.Migration{m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer result.Adapter.Close()

	if _, err := result.MigrationRunner.RunPending(); err != nil {
		t.Fatalf("RunPending failed: %v", err)
	}

	if !ran {
		t.Error("migration should have been executed")
	}
}

type testMigration struct {
	name string
	up   func(db ports.DBAdapter) error
}

func (m *testMigration) Name() string { return m.name }
func (m *testMigration) Up(db ports.DBAdapter) error {
	if m.up != nil {
		return m.up(db)
	}
	return nil
}
func (m *testMigration) Down(db ports.DBAdapter) error { return nil }
