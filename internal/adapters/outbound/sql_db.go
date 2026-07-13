package outbound

import (
	"database/sql"
)

// sqlDB is the storage core shared by the SQL-backed adapters (RFC-007). It
// owns the connection pool and defers backend differences to its dialect; the
// duplicated method bodies in the sqlite/postgres adapters migrate here group
// by group, written once with ?-placeholders and rebound per dialect.
type sqlDB struct {
	db      *sql.DB
	dialect dialect
}
