package outbound

import (
	"database/sql"
	"fmt"

	"fsos-server/internal/domain/ports"
)

// sqliteDialect adapts the storage core's shared SQL to SQLite.
type sqliteDialect struct{}

var _ dialect = sqliteDialect{}

func (sqliteDialect) Rebind(query string) string {
	return query
}

func (sqliteDialect) ColumnType(t ports.ColumnType) string {
	switch t {
	case ports.ColInteger:
		return "INTEGER"
	case ports.ColText:
		return "TEXT"
	case ports.ColDatetime:
		return "DATETIME"
	default:
		return "TEXT"
	}
}

func (sqliteDialect) NowExpr() string {
	return "datetime('now')"
}

func (sqliteDialect) DefaultNowExpr() string {
	return "(datetime('now'))"
}

func (sqliteDialect) DropTableSuffix() string {
	return ""
}

func (sqliteDialect) Init(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("failed to set %s: %w", p, err)
		}
	}
	return nil
}
