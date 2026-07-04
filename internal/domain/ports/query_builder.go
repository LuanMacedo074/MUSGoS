package ports

// QueryResult represents a single row as a map of column name to value.
type QueryResult map[string]interface{}

// QueryBuilder provides a fluent interface for generic table operations.
type QueryBuilder interface {
	Table(name string) Query
}

// Query represents a chainable query on a single table.
type Query interface {
	Where(column string, value interface{}) Query
	Insert(data map[string]interface{}) error
	Update(data map[string]interface{}) (int64, error)
	// Increment atomically adds delta to a numeric column (SET col = col + delta) in a
	// single UPDATE, so concurrent adjustments don't lose updates the way a
	// read-modify-write does. Returns the number of rows affected.
	Increment(column string, delta int64) (int64, error)
	Delete() (int64, error)
	First() (QueryResult, error)
	Get() ([]QueryResult, error)
	Count() (int64, error)
}

// Tx is a QueryBuilder whose operations run inside a single transaction, plus
// the means to finalize it. Table(name) returns queries bound to the tx.
type Tx interface {
	QueryBuilder
	Commit() error
	Rollback() error
}

// TransactionalQueryBuilder is a QueryBuilder that can open a transaction.
// Implementations that don't support transactions simply omit this interface;
// callers type-assert for it and degrade gracefully.
type TransactionalQueryBuilder interface {
	QueryBuilder
	Begin() (Tx, error)
}
