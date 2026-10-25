package services

import (
	"fmt"
	"sort"

	"fsos-server/internal/domain/ports"
)

type MigrationResult struct {
	Total   int
	Applied int
	Ran     []string
}

type MigrationRunner struct {
	tracker    ports.MigrationTracker
	db         ports.DBAdapter
	migrations []ports.Migration
}

func NewMigrationRunner(tracker ports.MigrationTracker, db ports.DBAdapter, migrations []ports.Migration) *MigrationRunner {
	sorted := make([]ports.Migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name() < sorted[j].Name()
	})

	return &MigrationRunner{
		tracker:    tracker,
		db:         db,
		migrations: sorted,
	}
}

func (r *MigrationRunner) RunPending() (MigrationResult, error) {
	result := MigrationResult{Total: len(r.migrations)}

	applied, err := r.tracker.GetAppliedMigrations()
	if err != nil {
		return result, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	result.Applied = len(applied)

	appliedSet := make(map[string]bool, len(applied))
	for _, name := range applied {
		appliedSet[name] = true
	}

	for _, m := range r.migrations {
		if appliedSet[m.Name()] {
			continue
		}

		if err := r.apply(m); err != nil {
			return result, err
		}

		result.Ran = append(result.Ran, m.Name())
	}

	return result, nil
}

// apply runs one migration and records it in the tracker. When the database
// supports it both happen in a single transaction, so a failing Up (or a crash
// before the tracker insert) leaves neither partial DDL nor a tracker row.
func (r *MigrationRunner) apply(m ports.Migration) error {
	transactor, ok := r.db.(ports.MigrationTransactor)
	if !ok {
		return runMigration(m, r.db, r.tracker)
	}

	tx, err := transactor.BeginMigration()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for migration %s: %w", m.Name(), err)
	}
	if err := runMigration(m, tx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", err, rbErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", m.Name(), err)
	}
	return nil
}

func runMigration(m ports.Migration, db ports.DBAdapter, tracker ports.MigrationTracker) error {
	if err := m.Up(db); err != nil {
		return fmt.Errorf("migration %s failed: %w", m.Name(), err)
	}
	if err := tracker.MarkMigrationApplied(m.Name()); err != nil {
		return fmt.Errorf("failed to mark migration %s as applied: %w", m.Name(), err)
	}
	return nil
}
