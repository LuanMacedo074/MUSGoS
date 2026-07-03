package outbound_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// createTxTestTable adds a simple k/v table for transaction tests.
func createTxTestTable(t *testing.T, db *outbound.SQLiteDB) {
	t.Helper()
	err := db.CreateTable(ports.Table{
		Name: "tx_test",
		Columns: []ports.Column{
			ports.Col("k", ports.ColText).NotNull(),
			ports.Col("v", ports.ColInteger).NotNull().Default(0),
		},
		PrimaryKeys: []string{"k"},
	})
	if err != nil {
		t.Fatalf("create tx_test: %v", err)
	}
}

func countTxTest(t *testing.T, qb ports.QueryBuilder) int64 {
	t.Helper()
	n, err := qb.Table("tx_test").Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// --- Go-level: Begin / Commit / Rollback -----------------------------------

func TestTx_CommitPersists(t *testing.T) {
	db := newTestDB(t)
	createTxTestTable(t, db)
	qb := db.QueryBuilder()

	txqb, ok := qb.(ports.TransactionalQueryBuilder)
	if !ok {
		t.Fatal("expected the sqlite query builder to be transactional")
	}
	tx, err := txqb.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := tx.Table("tx_test").Insert(map[string]interface{}{"k": "a", "v": 1}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := countTxTest(t, qb); got != 1 {
		t.Fatalf("after commit expected 1 row, got %d", got)
	}
}

func TestTx_RollbackDiscards(t *testing.T) {
	db := newTestDB(t)
	createTxTestTable(t, db)
	qb := db.QueryBuilder()

	tx, err := qb.(ports.TransactionalQueryBuilder).Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := tx.Table("tx_test").Insert(map[string]interface{}{"k": "a", "v": 1}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if got := countTxTest(t, qb); got != 0 {
		t.Fatalf("after rollback expected 0 rows, got %d", got)
	}
}

// --- Lua-level: mus.db.transaction ------------------------------------------

func newTxEngine(t *testing.T, dir string) (*outbound.SQLiteDB, ports.ScriptEngine) {
	t.Helper()
	db := newTestDB(t)
	createTxTestTable(t, db)
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, db, db.QueryBuilder(), nil, nil)
	return db, engine
}

func runInt(t *testing.T, engine ports.ScriptEngine, subject string) int64 {
	t.Helper()
	res, err := engine.Execute(&ports.ScriptMessage{Subject: subject, SenderID: "sys", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("execute %s: %v", subject, err)
	}
	n, ok := res.Content.(*lingo.LInteger)
	if !ok {
		t.Fatalf("execute %s: expected *LInteger, got %T", subject, res.Content)
	}
	return int64(n.Value)
}

func TestLuaTx_CommitsOnSuccess(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "commit", `
mus.db.transaction(function()
	mus.db.table("tx_test"):insert({k="a", v=1})
	mus.db.table("tx_test"):insert({k="b", v=2})
end)
mus.response(mus.db.table("tx_test"):count())
`)
	_, engine := newTxEngine(t, dir)
	if got := runInt(t, engine, "commit"); got != 2 {
		t.Fatalf("expected 2 committed rows, got %d", got)
	}
}

func TestLuaTx_RollsBackOnFalse(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "abort", `
mus.db.transaction(function()
	mus.db.table("tx_test"):insert({k="a", v=1})
	return false
end)
mus.response(mus.db.table("tx_test"):count())
`)
	_, engine := newTxEngine(t, dir)
	if got := runInt(t, engine, "abort"); got != 0 {
		t.Fatalf("expected 0 rows after return-false rollback, got %d", got)
	}
}

func TestLuaTx_RollsBackOnError(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "err", `
local ok = pcall(function()
	mus.db.transaction(function()
		mus.db.table("tx_test"):insert({k="a", v=1})
		error("boom")
	end)
end)
mus.response(mus.db.table("tx_test"):count())
`)
	_, engine := newTxEngine(t, dir)
	if got := runInt(t, engine, "err"); got != 0 {
		t.Fatalf("expected 0 rows after error rollback, got %d", got)
	}
}

func TestLuaTx_NestedRejected(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "nested", `
local ok = pcall(function()
	mus.db.transaction(function()
		mus.db.transaction(function() end)
	end)
end)
-- nested attempt must fail (ok=false); the outer tx rolls back its insert.
mus.response(ok and 1 or 0)
`)
	_, engine := newTxEngine(t, dir)
	if got := runInt(t, engine, "nested"); got != 0 {
		t.Fatalf("expected nested transaction to be rejected (0), got %d", got)
	}
}

// After a rolled-back transaction, mus.db.table(...) must route back to the
// root connection (active builder restored), not the dead tx.
func TestLuaTx_RestoresRootAfterRollback(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "restore", `
mus.db.transaction(function()
	mus.db.table("tx_test"):insert({k="a", v=1})
	return false
end)
-- this insert happens OUTSIDE any tx and must persist
mus.db.table("tx_test"):insert({k="b", v=2})
mus.response(mus.db.table("tx_test"):count())
`)
	_, engine := newTxEngine(t, dir)
	if got := runInt(t, engine, "restore"); got != 1 {
		t.Fatalf("expected 1 row (rolled-back a, persisted b), got %d", got)
	}
}
