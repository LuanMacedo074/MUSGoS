package outbound_test

import (
	"testing"

	"fsos-server/internal/domain/ports"
)

func TestCreateTable_ValidSchema(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "test_table",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.Col("name", ports.ColText).NotNull(),
		},
	})
	mustNoErr(t, err)
}

func TestCreateTable_InvalidTableName(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "DROP TABLE users; --",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
		},
	})
	if err == nil {
		t.Error("expected error for invalid table name")
	}
}

func TestCreateTable_InvalidColumnName(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "test_table",
		Columns: []ports.Column{
			ports.Col("col with spaces", ports.ColText),
		},
	})
	if err == nil {
		t.Error("expected error for invalid column name")
	}
}

func TestCreateTable_InvalidForeignKeyIdentifier(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "test_table",
		Columns: []ports.Column{
			ports.Col("ref_id", ports.ColInteger),
		},
		ForeignKeys: []ports.ForeignKey{
			{Column: "ref_id", RefTable: "other; DROP TABLE users", RefCol: "id", OnDelete: "CASCADE"},
		},
	})
	if err == nil {
		t.Error("expected error for invalid FK table reference")
	}
}

func TestCreateTable_InvalidOnDelete(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "test_table",
		Columns: []ports.Column{
			ports.Col("ref_id", ports.ColInteger),
		},
		ForeignKeys: []ports.ForeignKey{
			{Column: "ref_id", RefTable: "users", RefCol: "id", OnDelete: "CASCADE; DROP TABLE users"},
		},
	})
	if err == nil {
		t.Error("expected error for invalid ON DELETE action")
	}
}

func TestCreateTable_ValidOnDeleteActions(t *testing.T) {
	db := newTestDB(t)

	// Create a referenced table first
	mustNoErr(t, db.CreateTable(ports.Table{
		Name: "parent_table",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
		},
	}))

	for _, action := range []string{"CASCADE", "SET NULL", "RESTRICT", "NO ACTION", ""} {
		tableName := "child_" + action
		if action == "" {
			tableName = "child_empty"
		} else if action == "SET NULL" {
			tableName = "child_set_null"
		} else if action == "NO ACTION" {
			tableName = "child_no_action"
		}

		err := db.CreateTable(ports.Table{
			Name: tableName,
			Columns: []ports.Column{
				ports.PrimaryKey("id"),
				ports.Col("parent_id", ports.ColInteger),
			},
			ForeignKeys: []ports.ForeignKey{
				{Column: "parent_id", RefTable: "parent_table", RefCol: "id", OnDelete: action},
			},
		})
		if err != nil {
			t.Errorf("expected ON DELETE %q to be valid, got error: %v", action, err)
		}
	}
}

func TestDropTable_InvalidName(t *testing.T) {
	db := newTestDB(t)
	err := db.DropTable("users; DROP TABLE bans")
	if err == nil {
		t.Error("expected error for invalid table name in DropTable")
	}
}

func TestCreateIndex_InvalidIdentifiers(t *testing.T) {
	db := newTestDB(t)

	tests := []struct {
		name string
		idx  ports.Index
	}{
		{
			name: "invalid index name",
			idx:  ports.Index{Name: "idx; DROP TABLE users", Table: "users", Columns: []string{"id"}},
		},
		{
			name: "invalid table name",
			idx:  ports.Index{Name: "idx_test", Table: "users; --", Columns: []string{"id"}},
		},
		{
			name: "invalid column name",
			idx:  ports.Index{Name: "idx_test", Table: "users", Columns: []string{"id; DROP"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := db.CreateIndex(tc.idx)
			if err == nil {
				t.Error("expected error for invalid identifier")
			}
		})
	}
}

func TestCreateTable_CompositePK(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "composite_pk_table",
		Columns: []ports.Column{
			ports.Col("a", ports.ColInteger).NotNull(),
			ports.Col("b", ports.ColText).NotNull(),
			ports.Col("value", ports.ColText),
		},
		PrimaryKeys: []string{"a", "b"},
	})
	mustNoErr(t, err)
}

func TestCreateTable_InvalidPrimaryKeyColumn(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "test_table",
		Columns: []ports.Column{
			ports.Col("a", ports.ColInteger),
		},
		PrimaryKeys: []string{"a; DROP TABLE users"},
	})
	if err == nil {
		t.Error("expected error for invalid primary key column name")
	}
}

func TestCreateTable_RequireOneOf(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "check_table",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.Col("field_a", ports.ColText),
			ports.Col("field_b", ports.ColText),
		},
		RequireOneOf: []string{"field_a", "field_b"},
	})
	mustNoErr(t, err)
}

func TestCreateTable_InvalidRequireOneOf(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "test_table",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
		},
		RequireOneOf: []string{"field; DROP TABLE users"},
	})
	if err == nil {
		t.Error("expected error for invalid column in RequireOneOf")
	}
}

func TestCreateTable_DefaultLiteralWithQuotes(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateTable(ports.Table{
		Name: "quotes_table",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.Col("description", ports.ColText).NotNull().Default("it's a test"),
		},
	})
	mustNoErr(t, err)
}
