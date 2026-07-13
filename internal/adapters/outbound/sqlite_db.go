package outbound

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

// --- DBUser ---

// --- DBBan ---

// --- Schema operations ---

func (s *SQLiteDB) CreateTable(def ports.Table) error {
	if err := validateIdentifier(def.Name); err != nil {
		return fmt.Errorf("invalid table name %q: %w", def.Name, err)
	}
	for _, col := range def.Columns {
		if err := validateIdentifier(col.Name); err != nil {
			return fmt.Errorf("invalid column name %q: %w", col.Name, err)
		}
	}
	for _, fk := range def.ForeignKeys {
		for _, id := range []string{fk.Column, fk.RefTable, fk.RefCol} {
			if err := validateIdentifier(id); err != nil {
				return fmt.Errorf("invalid identifier %q in foreign key: %w", id, err)
			}
		}
		if !validOnDelete[fk.OnDelete] {
			return fmt.Errorf("invalid ON DELETE action %q", fk.OnDelete)
		}
	}
	for _, pk := range def.PrimaryKeys {
		if err := validateIdentifier(pk); err != nil {
			return fmt.Errorf("invalid primary key column %q: %w", pk, err)
		}
	}
	for _, col := range def.RequireOneOf {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid column %q in require_one_of: %w", col, err)
		}
	}

	var b strings.Builder
	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(def.Name)
	b.WriteString(" (\n")

	for i, col := range def.Columns {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("\t")
		b.WriteString(col.Name)
		b.WriteString(" ")
		b.WriteString(s.columnTypeSQL(col.Type))

		if col.IsPK && !s.hasCompositePK(def) {
			b.WriteString(" PRIMARY KEY")
			if col.IsAutoIncr {
				b.WriteString(" AUTOINCREMENT")
			}
		}
		if col.IsNotNull {
			b.WriteString(" NOT NULL")
		}
		if col.IsUnique {
			b.WriteString(" UNIQUE")
		}
		if col.DefType != ports.DefaultNone {
			b.WriteString(" DEFAULT ")
			b.WriteString(s.defaultSQL(col))
		}
	}

	if s.hasCompositePK(def) {
		b.WriteString(",\n\tPRIMARY KEY (")
		b.WriteString(strings.Join(def.PrimaryKeys, ", "))
		b.WriteString(")")
	}

	for _, fk := range def.ForeignKeys {
		b.WriteString(",\n\tFOREIGN KEY (")
		b.WriteString(fk.Column)
		b.WriteString(") REFERENCES ")
		b.WriteString(fk.RefTable)
		b.WriteString("(")
		b.WriteString(fk.RefCol)
		b.WriteString(")")
		if fk.OnDelete != "" {
			b.WriteString(" ON DELETE ")
			b.WriteString(fk.OnDelete)
		}
	}

	if len(def.RequireOneOf) > 0 {
		b.WriteString(",\n\tCHECK (")
		for i, col := range def.RequireOneOf {
			if i > 0 {
				b.WriteString(" OR ")
			}
			b.WriteString(col)
			b.WriteString(" IS NOT NULL")
		}
		b.WriteString(")")
	}

	b.WriteString("\n)")

	_, err := s.db.Exec(b.String())
	return err
}

func (s *SQLiteDB) DropTable(name string) error {
	if err := validateIdentifier(name); err != nil {
		return fmt.Errorf("invalid table name %q: %w", name, err)
	}
	_, err := s.db.Exec("DROP TABLE IF EXISTS " + name)
	return err
}

func (s *SQLiteDB) AddColumn(table string, col ports.Column) error {
	if err := validateIdentifier(table); err != nil {
		return fmt.Errorf("invalid table name %q: %w", table, err)
	}
	if err := validateIdentifier(col.Name); err != nil {
		return fmt.Errorf("invalid column name %q: %w", col.Name, err)
	}
	// SQLite has no ADD COLUMN IF NOT EXISTS; skip when the column already exists so
	// re-running a migration is a no-op.
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == col.Name {
			rows.Close()
			return nil
		}
	}
	rows.Close()

	var b strings.Builder
	b.WriteString("ALTER TABLE ")
	b.WriteString(table)
	b.WriteString(" ADD COLUMN ")
	b.WriteString(col.Name)
	b.WriteString(" ")
	b.WriteString(s.columnTypeSQL(col.Type))
	if col.IsNotNull {
		b.WriteString(" NOT NULL")
	}
	if col.DefType != ports.DefaultNone {
		b.WriteString(" DEFAULT ")
		b.WriteString(s.defaultSQL(col))
	}
	_, err = s.db.Exec(b.String())
	return err
}

func (s *SQLiteDB) CreateIndex(def ports.Index) error {
	if err := validateIdentifier(def.Name); err != nil {
		return fmt.Errorf("invalid index name %q: %w", def.Name, err)
	}
	if err := validateIdentifier(def.Table); err != nil {
		return fmt.Errorf("invalid table name %q: %w", def.Table, err)
	}
	for _, col := range def.Columns {
		if err := validateIdentifier(col); err != nil {
			return fmt.Errorf("invalid column name %q: %w", col, err)
		}
	}
	sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
		def.Name, def.Table, strings.Join(def.Columns, ", "))
	_, err := s.db.Exec(sql)
	return err
}

func (s *SQLiteDB) columnTypeSQL(t ports.ColumnType) string {
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

func (s *SQLiteDB) defaultSQL(col ports.Column) string {
	switch col.DefType {
	case ports.DefaultNow:
		return "(datetime('now'))"
	case ports.DefaultLiteral:
		switch v := col.DefaultVal.(type) {
		case string:
			escaped := strings.ReplaceAll(v, "'", "''")
			return fmt.Sprintf("'%s'", escaped)
		default:
			return fmt.Sprintf("%v", v)
		}
	default:
		return ""
	}
}

func (s *SQLiteDB) hasCompositePK(def ports.Table) bool {
	return len(def.PrimaryKeys) > 0
}

// QueryBuilder returns a generic query builder for this database.
func (s *SQLiteDB) QueryBuilder() ports.QueryBuilder {
	return NewSQLiteQueryBuilder(s.db)
}

// --- Close ---

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}
