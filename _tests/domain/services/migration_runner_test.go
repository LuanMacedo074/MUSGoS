package services_test

import (
	"fmt"
	"testing"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
)

// --- fake MigrationTracker ---

type fakeMigrationTracker struct {
	applied []string
}

func (f *fakeMigrationTracker) GetAppliedMigrations() ([]string, error) {
	return f.applied, nil
}

func (f *fakeMigrationTracker) MarkMigrationApplied(name string) error {
	f.applied = append(f.applied, name)
	return nil
}

// --- fake Migration ---

type fakeMigration struct {
	name    string
	upCalls int
	upErr   error
}

func (m *fakeMigration) Name() string                   { return m.name }
func (m *fakeMigration) Up(db ports.DBAdapter) error     { m.upCalls++; return m.upErr }
func (m *fakeMigration) Down(db ports.DBAdapter) error   { return nil }

func TestMigrationRunner_RunPending_AllNew(t *testing.T) {
	tracker := &fakeMigrationTracker{}
	m1 := &fakeMigration{name: "20260101_first"}
	m2 := &fakeMigration{name: "20260102_second"}

	runner := services.NewMigrationRunner(tracker, nil, []ports.Migration{m2, m1})

	if err := runner.RunPending(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m1.upCalls != 1 {
		t.Errorf("expected m1 Up called once, got %d", m1.upCalls)
	}
	if m2.upCalls != 1 {
		t.Errorf("expected m2 Up called once, got %d", m2.upCalls)
	}
	if len(tracker.applied) != 2 {
		t.Fatalf("expected 2 applied, got %d", len(tracker.applied))
	}
	// Should be sorted by name
	if tracker.applied[0] != "20260101_first" {
		t.Errorf("expected first applied to be m1, got %q", tracker.applied[0])
	}
	if tracker.applied[1] != "20260102_second" {
		t.Errorf("expected second applied to be m2, got %q", tracker.applied[1])
	}
}

func TestMigrationRunner_RunPending_SkipsApplied(t *testing.T) {
	tracker := &fakeMigrationTracker{
		applied: []string{"20260101_first"},
	}
	m1 := &fakeMigration{name: "20260101_first"}
	m2 := &fakeMigration{name: "20260102_second"}

	runner := services.NewMigrationRunner(tracker, nil, []ports.Migration{m1, m2})

	if err := runner.RunPending(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m1.upCalls != 0 {
		t.Errorf("m1 should have been skipped, got %d calls", m1.upCalls)
	}
	if m2.upCalls != 1 {
		t.Errorf("m2 should have run, got %d calls", m2.upCalls)
	}
}

func TestMigrationRunner_RunPending_NoPending(t *testing.T) {
	tracker := &fakeMigrationTracker{
		applied: []string{"20260101_first"},
	}
	m1 := &fakeMigration{name: "20260101_first"}

	runner := services.NewMigrationRunner(tracker, nil, []ports.Migration{m1})

	if err := runner.RunPending(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m1.upCalls != 0 {
		t.Error("should not have called Up on already-applied migration")
	}
}

func TestMigrationRunner_RunPending_UpError(t *testing.T) {
	tracker := &fakeMigrationTracker{}
	m1 := &fakeMigration{name: "20260101_first", upErr: fmt.Errorf("schema error")}

	runner := services.NewMigrationRunner(tracker, nil, []ports.Migration{m1})

	err := runner.RunPending()
	if err == nil {
		t.Fatal("expected error when migration fails")
	}

	if len(tracker.applied) != 0 {
		t.Error("failed migration should not be marked as applied")
	}
}

func TestMigrationRunner_RunPending_Empty(t *testing.T) {
	tracker := &fakeMigrationTracker{}
	runner := services.NewMigrationRunner(tracker, nil, []ports.Migration{})

	if err := runner.RunPending(); err != nil {
		t.Fatalf("unexpected error with no migrations: %v", err)
	}
}
