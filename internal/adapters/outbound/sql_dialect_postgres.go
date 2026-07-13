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

func (postgresDialect) Init(db *sql.DB) error {
	return nil
}
