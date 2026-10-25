package ports

// ColumnType represents the abstract type of a database column.
type ColumnType int

const (
	ColInteger ColumnType = iota
	ColText
	ColDatetime
	ColJSONB // structured JSON; native JSONB on Postgres, TEXT on SQLite
)

// DefaultType represents how a column's default value is determined.
type DefaultType int

const (
	DefaultNone    DefaultType = iota
	DefaultNow                 // current timestamp
	DefaultLiteral             // literal value from DefaultVal
)

// Column describes a single column in a table definition.
type Column struct {
	Name       string
	Type       ColumnType
	IsNotNull  bool
	IsUnique   bool
	IsPK       bool
	IsAutoIncr bool
	DefType    DefaultType
	DefaultVal interface{}
}

// Col creates a new column with the given name and type.
func Col(name string, t ColumnType) Column {
	return Column{Name: name, Type: t}
}

// NotNull marks the column as NOT NULL.
func (c Column) NotNull() Column { c.IsNotNull = true; return c }

// Unique marks the column as UNIQUE.
func (c Column) Unique() Column { c.IsUnique = true; return c }

// Default sets a literal default value.
func (c Column) Default(v interface{}) Column {
	c.DefType = DefaultLiteral
	c.DefaultVal = v
	return c
}

// DefaultNow sets the default to the current timestamp.
func (c Column) DefaultNow() Column { c.DefType = DefaultNow; return c }

// PrimaryKey creates an integer auto-increment primary key column.
func PrimaryKey(name string) Column {
	return Column{
		Name:       name,
		Type:       ColInteger,
		IsPK:       true,
		IsAutoIncr: true,
		IsNotNull:  true,
	}
}

// UUID creates a text column with NOT NULL and UNIQUE.
// UUID values should be generated in Go (e.g. google/uuid) and passed explicitly.
func UUID(name string) Column {
	return Column{
		Name:      name,
		Type:      ColText,
		IsNotNull: true,
		IsUnique:  true,
	}
}

// ForeignKey defines a foreign key constraint.
type ForeignKey struct {
	Column   string
	RefTable string
	RefCol   string
	OnDelete string
}

// RangeCheck is a CHECK constraint bounding an integer column to an inclusive
// range: Min <= Column <= Max. It lets a schema cap a value (e.g. a slot
// number) that the column type alone cannot.
type RangeCheck struct {
	Column string
	Min    int
	Max    int
}

// InCheck is a CHECK constraint restricting a text column to a fixed set of
// values: Column IN (Values...). It lets a schema enforce a closed enumeration
// the column type alone cannot. Values are emitted as SQL string literals, so
// each must be a plain identifier-like word (validated before any SQL is built).
type InCheck struct {
	Column string
	Values []string
}

// Table defines a table to be created.
type Table struct {
	Name         string
	Columns      []Column
	PrimaryKeys  []string // composite primary key columns
	ForeignKeys  []ForeignKey
	RequireOneOf []string     // CHECK: at least one of these must be non-null
	RangeChecks  []RangeCheck // CHECK: each column within its inclusive [Min,Max]
	InChecks     []InCheck    // CHECK: each column one of its listed values
}

// Index defines an index to be created.
type Index struct {
	Name    string
	Table   string
	Columns []string
}
