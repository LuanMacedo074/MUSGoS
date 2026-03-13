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
	Delete() (int64, error)
	First() (QueryResult, error)
	Get() ([]QueryResult, error)
	Count() (int64, error)
}
