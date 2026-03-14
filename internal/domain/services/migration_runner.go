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

		if err := m.Up(r.db); err != nil {
			return result, fmt.Errorf("migration %s failed: %w", m.Name(), err)
		}

		if err := r.tracker.MarkMigrationApplied(m.Name()); err != nil {
			return result, fmt.Errorf("failed to mark migration %s as applied: %w", m.Name(), err)
		}

		result.Ran = append(result.Ran, m.Name())
	}

	return result, nil
}
