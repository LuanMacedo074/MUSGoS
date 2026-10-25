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

func TestDropColumn_RemovesColumn(t *testing.T) {
	db := newTestDB(t)

	mustNoErr(t, db.CreateTable(ports.Table{
		Name: "drop_test",
		Columns: []ports.Column{
			ports.Col("k", ports.ColText).NotNull(),
			ports.Col("keep", ports.ColInteger).NotNull().Default(0),
			ports.Col("goner", ports.ColInteger).NotNull().Default(0),
		},
		PrimaryKeys: []string{"k"},
	}))

	qb := db.QueryBuilder()
	mustNoErr(t, qb.Table("drop_test").Insert(map[string]interface{}{"k": "a", "keep": 1, "goner": 9}))

	mustNoErr(t, db.DropColumn("drop_test", "goner"))

	row, err := qb.Table("drop_test").Where("k", "a").First()
	mustNoErr(t, err)
	if row == nil {
		t.Fatal("expected the row to survive the column drop")
	}
	if _, ok := row["goner"]; ok {
		t.Error("dropped column 'goner' should no longer appear in a selected row")
	}
	if _, ok := row["keep"]; !ok {
		t.Error("surviving column 'keep' should still be present")
	}

	// Dropping an already-absent column is a no-op, mirroring AddColumn's
	// idempotent re-run so migrations stay re-runnable.
	mustNoErr(t, db.DropColumn("drop_test", "goner"))
}

func TestDropColumn_InvalidIdentifiers(t *testing.T) {
	db := newTestDB(t)
	if err := db.DropColumn("users; DROP TABLE bans", "x"); err == nil {
		t.Error("expected error for invalid table name in DropColumn")
	}
	if err := db.DropColumn("users", "col; DROP TABLE bans"); err == nil {
		t.Error("expected error for invalid column name in DropColumn")
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

func TestCreateTable_RangeCheckBoundsColumn(t *testing.T) {
	assertRangeCheckBoundsColumn(t, newTestDB(t))
}

func TestCreateTable_InCheckRestrictsColumn(t *testing.T) {
	assertInCheckRestrictsColumn(t, newTestDB(t))
}

func TestCreateTable_InvalidRangeCheck(t *testing.T) {
	cases := map[string]ports.RangeCheck{
		"invalid column": {Column: "slot; DROP TABLE users", Min: 1, Max: 5},
		"inverted range": {Column: "slot", Min: 5, Max: 1},
	}
	for name, check := range cases {
		t.Run(name, func(t *testing.T) {
			db := newTestDB(t)
			err := db.CreateTable(ports.Table{
				Name: "test_table",
				Columns: []ports.Column{
					ports.PrimaryKey("id"),
					ports.Col("slot", ports.ColInteger),
				},
				RangeChecks: []ports.RangeCheck{check},
			})
			if err == nil {
				t.Error("expected error for an invalid RangeCheck")
			}
		})
	}
}

func TestCreateTable_InvalidInCheck(t *testing.T) {
	cases := map[string]ports.InCheck{
		"invalid column": {Column: "kind; DROP TABLE users", Values: []string{"a"}},
		"no values":      {Column: "kind"},
		"quoted value":   {Column: "kind", Values: []string{"a') OR ('1'='1"}},
	}
	for name, check := range cases {
		t.Run(name, func(t *testing.T) {
			db := newTestDB(t)
			err := db.CreateTable(ports.Table{
				Name: "test_table",
				Columns: []ports.Column{
					ports.PrimaryKey("id"),
					ports.Col("kind", ports.ColText),
				},
				InChecks: []ports.InCheck{check},
			})
			if err == nil {
				t.Error("expected error for an invalid InCheck")
			}
		})
	}
}

// assertRangeCheckBoundsColumn checks that a RangeCheck emits an inclusive
// bound: both ends accepted, one past either end rejected. The Postgres
// test reset drops check_test, so the table never outlives a run.
func assertRangeCheckBoundsColumn(t *testing.T, db predicateDB) {
	t.Helper()
	mustNoErr(t, db.CreateTable(ports.Table{
		Name: "check_test",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.Col("slot", ports.ColInteger).NotNull(),
		},
		RangeChecks: []ports.RangeCheck{{Column: "slot", Min: 1, Max: 5}},
	}))

	query := db.QueryBuilder().Table("check_test")
	for _, slot := range []int{1, 5} {
		if err := query.Insert(map[string]interface{}{"slot": slot}); err != nil {
			t.Errorf("insert slot=%d (in range): %v", slot, err)
		}
	}
	for _, slot := range []int{0, 6} {
		if err := query.Insert(map[string]interface{}{"slot": slot}); err == nil {
			t.Errorf("insert slot=%d (out of range) succeeded, want a CHECK rejection", slot)
		}
	}
}

// assertInCheckRestrictsColumn checks that an InCheck admits only its listed
// values.
func assertInCheckRestrictsColumn(t *testing.T, db predicateDB) {
	t.Helper()
	mustNoErr(t, db.CreateTable(ports.Table{
		Name: "check_test",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.Col("kind", ports.ColText).NotNull(),
		},
		InChecks: []ports.InCheck{{Column: "kind", Values: []string{"alpha", "beta"}}},
	}))

	query := db.QueryBuilder().Table("check_test")
	for _, kind := range []string{"alpha", "beta"} {
		if err := query.Insert(map[string]interface{}{"kind": kind}); err != nil {
			t.Errorf("insert kind=%q (listed): %v", kind, err)
		}
	}
	if err := query.Insert(map[string]interface{}{"kind": "gamma"}); err == nil {
		t.Error("insert kind=\"gamma\" (unlisted) succeeded, want a CHECK rejection")
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
