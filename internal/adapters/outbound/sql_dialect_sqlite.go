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
	case ports.ColJSONB:
		// SQLite has no JSONB storage type; store as TEXT and use json1 funcs.
		return "TEXT"
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

func (sqliteDialect) AutoIncrPKSQL() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (sqliteDialect) AddColumnClause() string {
	return "ADD COLUMN "
}

// SkipAddColumn checks PRAGMA table_info because SQLite has no ADD COLUMN IF
// NOT EXISTS; skipping when the column exists keeps migrations re-runnable.
func (sqliteDialect) SkipAddColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (sqliteDialect) SupportsUniqueInAddColumn() bool {
	return false
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
