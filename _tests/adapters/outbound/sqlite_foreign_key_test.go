package outbound_test

import (
	"path/filepath"
	"testing"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

// On SQLite `foreign_keys` is a PER-CONNECTION pragma, so `sqliteDialect.Init`'s single
// `PRAGMA foreign_keys = ON` binds only the one pooled connection it ran on; a DELETE served
// by any other connection would skip ON DELETE CASCADE and orphan child rows. NewSQLiteDB
// closes that by opening with the DSN `?_pragma=foreign_keys(1)`, which modernc applies to
// every connection it opens.
//
// The test pins the fix through the adapter: it holds a transaction open to keep Init's
// connection checked out, so the cascade DELETE is forced onto a different pooled connection.
// Without the DSN pragma the child row survives because that connection never had
// foreign_keys turned on.
func TestSQLite_ForeignKeyCascadeOnNonInitConnection(t *testing.T) {
	db, err := outbound.NewSQLiteDB(filepath.Join(t.TempDir(), "fk.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	mustNoErr(t, db.CreateTable(ports.Table{
		Name:        "fk_parent",
		Columns:     []ports.Column{ports.Col("name", ports.ColText).NotNull()},
		PrimaryKeys: []string{"name"},
	}))
	mustNoErr(t, db.CreateTable(ports.Table{
		Name: "fk_child",
		Columns: []ports.Column{
			ports.Col("owner", ports.ColText).NotNull(),
			ports.Col("slot", ports.ColInteger).NotNull(),
		},
		PrimaryKeys: []string{"owner", "slot"},
		ForeignKeys: []ports.ForeignKey{
			{Column: "owner", RefTable: "fk_parent", RefCol: "name", OnDelete: "CASCADE"},
		},
	}))
	mustNoErr(t, db.QueryBuilder().Table("fk_parent").Insert(map[string]interface{}{"name": "p"}))
	mustNoErr(t, db.QueryBuilder().Table("fk_child").Insert(map[string]interface{}{"owner": "p", "slot": 1}))

	// Pin the (single, idle) connection Init ran the PRAGMA on by opening a transaction on it,
	// so the DELETE below must run on a freshly opened, distinct connection.
	txqb, ok := db.QueryBuilder().(ports.TransactionalQueryBuilder)
	if !ok {
		t.Fatal("sqlite QueryBuilder must support transactions to pin a connection")
	}
	tx, err := txqb.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := db.QueryBuilder().Table("fk_parent").Where("name", "p").Delete(); err != nil {
		t.Fatalf("delete parent: %v", err)
	}

	n, err := db.QueryBuilder().Table("fk_child").Where("owner", "p").Count()
	if err != nil {
		t.Fatalf("count children: %v", err)
	}
	if n != 0 {
		t.Errorf("child rows after cascade delete on a non-Init connection = %d, want 0 "+
			"(FK not enforced on that connection: the NewSQLiteDB DSN pragma is missing)", n)
	}
}
