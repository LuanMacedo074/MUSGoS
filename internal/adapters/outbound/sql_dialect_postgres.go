package outbound

import (
	"database/sql"

	"fsos-server/internal/domain/ports"
)

// postgresDialect adapts the storage core's shared SQL to PostgreSQL.
type postgresDialect struct{}

var _ dialect = postgresDialect{}

func (postgresDialect) Rebind(query string) string {
	return RebindDollar(query)
}

func (postgresDialect) ColumnType(t ports.ColumnType) string {
	switch t {
	case ports.ColInteger:
		return "BIGINT"
	case ports.ColText:
		return "TEXT"
	case ports.ColDatetime:
		return "TIMESTAMPTZ"
	case ports.ColJSONB:
		return "JSONB"
	default:
		return "TEXT"
	}
}

func (postgresDialect) NowExpr() string {
	return "now()"
}

func (postgresDialect) DefaultNowExpr() string {
	return "now()"
}

func (postgresDialect) DropTableSuffix() string {
	return " CASCADE"
}

func (postgresDialect) AutoIncrPKSQL() string {
	return "BIGSERIAL PRIMARY KEY"
}

func (postgresDialect) AddColumnClause() string {
	return "ADD COLUMN IF NOT EXISTS "
}

func (postgresDialect) SkipAddColumn(db *sql.DB, table, column string) (bool, error) {
	return false, nil // the IF NOT EXISTS clause handles idempotency
}

func (postgresDialect) SupportsUniqueInAddColumn() bool {
	return true
}

func (postgresDialect) Init(db *sql.DB) error {
	return nil
}
