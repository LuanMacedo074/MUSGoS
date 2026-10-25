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

func (m *fakeMigration) Name() string                  { return m.name }
func (m *fakeMigration) Up(db ports.DBAdapter) error   { m.upCalls++; return m.upErr }
func (m *fakeMigration) Down(db ports.DBAdapter) error { return nil }

func TestMigrationRunner_RunPending_AllNew(t *testing.T) {
	tracker := &fakeMigrationTracker{}
	m1 := &fakeMigration{name: "20260101_first"}
	m2 := &fakeMigration{name: "20260102_second"}

	runner := services.NewMigrationRunner(tracker, nil, []ports.Migration{m2, m1})

	if _, err := runner.RunPending(); err != nil {
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

	if _, err := runner.RunPending(); err != nil {
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

	if _, err := runner.RunPending(); err != nil {
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

	_, err := runner.RunPending()
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

	if _, err := runner.RunPending(); err != nil {
		t.Fatalf("unexpected error with no migrations: %v", err)
	}
}

// --- fake transactional DB ---

// fakeMigrationTx records what ran inside it and whether it was finalized.
// Embedding the interfaces leaves the unused DBAdapter methods nil — the
// runner only needs the tracker half and Commit/Rollback.
type fakeMigrationTx struct {
	ports.DBAdapter
	marked     []string
	markErr    error
	committed  bool
	rolledBack bool
}

func (tx *fakeMigrationTx) GetAppliedMigrations() ([]string, error) { return nil, nil }
func (tx *fakeMigrationTx) MarkMigrationApplied(name string) error {
	if tx.markErr != nil {
		return tx.markErr
	}
	tx.marked = append(tx.marked, name)
	return nil
}
func (tx *fakeMigrationTx) Commit() error   { tx.committed = true; return nil }
func (tx *fakeMigrationTx) Rollback() error { tx.rolledBack = true; return nil }

// fakeTransactionalDB is a transactional adapter that is also its own root
// tracker, as the SQL adapters are.
type fakeTransactionalDB struct {
	ports.DBAdapter
	fakeMigrationTracker
	txs     []*fakeMigrationTx
	markErr error
}

func (db *fakeTransactionalDB) BeginMigration() (ports.MigrationTx, error) {
	tx := &fakeMigrationTx{markErr: db.markErr}
	db.txs = append(db.txs, tx)
	return tx, nil
}

// recordingMigration remembers the adapter its Up received.
type recordingMigration struct {
	fakeMigration
	gotDB ports.DBAdapter
}

func (m *recordingMigration) Up(db ports.DBAdapter) error {
	m.gotDB = db
	return m.fakeMigration.Up(db)
}

func TestMigrationRunner_Transactional_EachMigrationCommitsWithItsTrackerRow(t *testing.T) {
	db := &fakeTransactionalDB{}
	m1 := &recordingMigration{fakeMigration: fakeMigration{name: "20260101_first"}}
	m2 := &recordingMigration{fakeMigration: fakeMigration{name: "20260102_second"}}

	runner := services.NewMigrationRunner(db, db, []ports.Migration{m2, m1})
	res, err := runner.RunPending()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(db.txs) != 2 {
		t.Fatalf("expected one transaction per migration (2), got %d", len(db.txs))
	}
	for i, want := range []string{"20260101_first", "20260102_second"} {
		tx := db.txs[i]
		if len(tx.marked) != 1 || tx.marked[0] != want {
			t.Errorf("tx %d: expected tracker row %q inside the tx, got %v", i, want, tx.marked)
		}
		if !tx.committed || tx.rolledBack {
			t.Errorf("tx %d: expected commit only, got committed=%v rolledBack=%v", i, tx.committed, tx.rolledBack)
		}
	}
	if m1.gotDB != db.txs[0] || m2.gotDB != db.txs[1] {
		t.Error("each migration's Up must run against its own transaction")
	}
	if len(db.applied) != 0 {
		t.Errorf("tracker rows must be written through the transaction, not the root tracker; got %v", db.applied)
	}
	if len(res.Ran) != 2 {
		t.Errorf("expected 2 ran, got %v", res.Ran)
	}
}

func TestMigrationRunner_Transactional_UpErrorRollsBack(t *testing.T) {
	db := &fakeTransactionalDB{}
	m1 := &fakeMigration{name: "20260101_first", upErr: fmt.Errorf("schema error")}
	m2 := &fakeMigration{name: "20260102_second"}

	runner := services.NewMigrationRunner(db, db, []ports.Migration{m1, m2})
	res, err := runner.RunPending()
	if err == nil {
		t.Fatal("expected error when migration fails")
	}

	if len(db.txs) != 1 {
		t.Fatalf("runner must stop at the failing migration; got %d txs", len(db.txs))
	}
	tx := db.txs[0]
	if !tx.rolledBack || tx.committed {
		t.Errorf("expected rollback only, got committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
	if len(tx.marked) != 0 {
		t.Errorf("failed migration must not be marked applied, got %v", tx.marked)
	}
	if m2.upCalls != 0 || len(res.Ran) != 0 {
		t.Error("no migration after the failing one may run")
	}
}

func TestMigrationRunner_Transactional_MarkErrorRollsBack(t *testing.T) {
	db := &fakeTransactionalDB{markErr: fmt.Errorf("duplicate")}
	m1 := &fakeMigration{name: "20260101_first"}

	runner := services.NewMigrationRunner(db, db, []ports.Migration{m1})
	if _, err := runner.RunPending(); err == nil {
		t.Fatal("expected error when the tracker insert fails")
	}

	tx := db.txs[0]
	if !tx.rolledBack || tx.committed {
		t.Errorf("a failed tracker insert must roll back the migration's DDL; committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
}

func TestMigrationRunner_Transactional_UpPanicRollsBack(t *testing.T) {
	db := &fakeTransactionalDB{}
	m1 := &panickingMigration{name: "20260101_first"}

	runner := services.NewMigrationRunner(db, db, []ports.Migration{m1})
	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic must propagate to the caller")
			}
		}()
		runner.RunPending()
	}()

	if len(db.txs) != 1 || !db.txs[0].rolledBack || db.txs[0].committed {
		t.Error("a panicking Up must roll back its transaction instead of leaving it open")
	}
}

type panickingMigration struct{ name string }

func (m *panickingMigration) Name() string                  { return m.name }
func (m *panickingMigration) Up(db ports.DBAdapter) error   { panic("boom") }
func (m *panickingMigration) Down(db ports.DBAdapter) error { return nil }

func TestMigrationRunner_SeparateTrackerSkipsTransaction(t *testing.T) {
	tracker := &fakeMigrationTracker{}
	db := &fakeTransactionalDB{}
	m1 := &fakeMigration{name: "20260101_first"}

	runner := services.NewMigrationRunner(tracker, db, []ports.Migration{m1})
	if _, err := runner.RunPending(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(db.txs) != 0 {
		t.Error("a transaction on db cannot cover a tracker that lives elsewhere; expected the two-step path")
	}
	if len(tracker.applied) != 1 || tracker.applied[0] != "20260101_first" {
		t.Errorf("the row must land in the tracker the runner reads from, got %v", tracker.applied)
	}
}
