package outbound

import (
	"database/sql"
	"strconv"
	"strings"

	"fsos-server/internal/domain/ports"
)

// dialect captures everything that differs between the SQL backends so the
// storage core can be written once (RFC-007). It is a seam internal to this
// package: nothing outside internal/adapters/outbound may depend on it, and
// there is exactly one production implementation per backend — no mocks.
type dialect interface {
	// Rebind converts a query written with ?-style placeholders into the
	// backend's placeholder style. All shared SQL is written with ?.
	Rebind(query string) string
	// ColumnType maps an abstract schema column type to the backend's SQL type.
	ColumnType(t ports.ColumnType) string
	// NowExpr is the current-timestamp expression usable inline in a statement.
	NowExpr() string
	// DefaultNowExpr is the current-timestamp expression usable in a column
	// DEFAULT clause (SQLite requires expression defaults to be parenthesized).
	DefaultNowExpr() string
	// DropTableSuffix is appended to DROP TABLE statements (" CASCADE" on
	// Postgres, empty on SQLite).
	DropTableSuffix() string
	// AutoIncrPKSQL is the full column definition fragment (type + constraints)
	// for a single-column auto-increment primary key.
	AutoIncrPKSQL() string
	// Init applies per-connection setup right after the pool is opened
	// (pragmas on SQLite; nothing on Postgres).
	Init(db *sql.DB) error
}

// RebindDollar rewrites ?-style placeholders as $1..$N, leaving question marks
// inside single-quoted string literals untouched ('' escapes a quote within a
// literal). It backs the Postgres dialect's Rebind and is exported only so the
// unit suite in _tests can exercise it directly.
func RebindDollar(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 8)

	n := 0
	inLiteral := false
	for _, r := range query {
		switch {
		case r == '\'':
			// A '' inside a literal toggles out and straight back in, which
			// leaves inLiteral correct without lookahead.
			inLiteral = !inLiteral
			b.WriteRune(r)
		case r == '?' && !inLiteral:
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
