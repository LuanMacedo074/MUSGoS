package outbound

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"fsos-server/internal/domain/ports"

	_ "modernc.org/sqlite"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

var validOnDelete = map[string]bool{
	"":            true,
	"CASCADE":     true,
	"SET NULL":    true,
	"SET DEFAULT": true,
	"RESTRICT":    true,
	"NO ACTION":   true,
}

func validateIdentifier(name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("identifier must match [a-zA-Z_][a-zA-Z0-9_]*")
	}
	return nil
}

type SQLiteDB struct {
	sqlDB
}

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	s := &SQLiteDB{sqlDB{db: db, dialect: sqliteDialect{}}}
	if err := s.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return s, nil
}

func (s *SQLiteDB) init() error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
	}
	for _, p := range pragmas {
		if _, err := s.db.Exec(p); err != nil {
			return fmt.Errorf("failed to set %s: %w", p, err)
		}
	}

	return s.ensureMigrationsTable()
}

// QueryBuilder returns a generic query builder for this database.
func (s *SQLiteDB) QueryBuilder() ports.QueryBuilder {
	return NewSQLiteQueryBuilder(s.db)
}

// --- Close ---

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}
