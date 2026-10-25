package outbound

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"fsos-server/internal/domain/ports"

	_ "modernc.org/sqlite"
)

// SQLiteDB implements ports.DBAdapter and ports.MigrationTracker against
// SQLite. All persistence logic lives on the embedded storage core; this type
// only owns construction (file path handling).
type SQLiteDB struct {
	sqlDB
}

// The migration runner only applies migrations atomically on adapters that
// implement ports.MigrationTransactor; fail the build if this one stops.
var _ ports.MigrationTransactor = (*SQLiteDB)(nil)

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// foreign_keys is a PER-CONNECTION pragma: a single `PRAGMA foreign_keys = ON`
	// (see sqliteDialect.Init) binds only whichever pooled connection ran it, so a
	// DELETE served by any other connection would skip ON DELETE CASCADE and orphan
	// child rows. The DSN `_pragma` applies it to EVERY connection modernc opens, so
	// FK enforcement holds across the whole pool.
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	s := &SQLiteDB{newSQLDB(db, sqliteDialect{})}
	if err := s.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return s, nil
}
