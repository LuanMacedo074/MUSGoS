package outbound_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"fsos-server/external/migrations"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
	"fsos-server/internal/domain/types/lingo"
)

// Postgres tests are integration tests: they need a live server reachable via
// TEST_POSTGRES_DSN (e.g. postgres://postgres:pass@localhost:5432/postgres?sslmode=disable).
// When it is unset the whole file skips, so `make test` stays green without one.
//
// WARNING: the target database is reset (its schema tables are dropped) on every
// test — point TEST_POSTGRES_DSN at a throwaway database, never a real one.

func newTestPG(t *testing.T) *outbound.PostgresDB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping postgres integration tests")
	}

	db, err := outbound.NewPostgresDB(dsn)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	resetPGSchema(t, db)

	runner := services.NewMigrationRunner(db, db, migrations.All)
	if _, err := runner.RunPending(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// resetPGSchema drops every table this suite may have created and clears the
// migration ledger so the migrations re-run against a clean slate.
func resetPGSchema(t *testing.T, db *outbound.PostgresDB) {
	t.Helper()
	for _, tbl := range []string{
		"tx_test", "bans", "player_attributes", "application_attributes", "users", "applications",
	} {
		if err := db.DropTable(tbl); err != nil {
			t.Fatalf("failed to drop %s: %v", tbl, err)
		}
	}
	if _, err := db.QueryBuilder().Table("migrations").Delete(); err != nil {
		t.Fatalf("failed to clear migrations ledger: %v", err)
	}
}

// --- DBAdmin / applications ---

func TestPG_CreateApplication_Duplicate(t *testing.T) {
	db := newTestPG(t)
	mustNoErr(t, db.CreateApplication("testApp"))
	if err := db.CreateApplication("testApp"); err == nil {
		t.Error("expected error for duplicate application")
	}
}

func TestPG_DeleteApplication_CascadesAttributes(t *testing.T) {
	db := newTestPG(t)
	mustNoErr(t, db.CreateApplication("app1"))
	mustNoErr(t, db.SetApplicationAttribute("app1", "key", lingo.NewLInteger(1)))
	mustNoErr(t, db.SetPlayerAttribute("app1", "user1", "score", lingo.NewLInteger(100)))
	mustNoErr(t, db.DeleteApplication("app1"))

	mustNoErr(t, db.CreateApplication("app1"))
	got, err := db.GetApplicationAttribute("app1", "key")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtVoid {
		t.Error("application attributes should have cascaded on delete")
	}
	got2, err := db.GetPlayerAttribute("app1", "user1", "score")
	mustNoErr(t, err)
	if got2.GetType() != lingo.VtVoid {
		t.Error("player attributes should have cascaded on delete")
	}
}

// --- attributes ---

func TestPG_ApplicationAttribute_SetGetOverwrite(t *testing.T) {
	db := newTestPG(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "val", lingo.NewLInteger(42)))
	got, err := db.GetApplicationAttribute("app1", "val")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtInteger || got.ToInteger() != 42 {
		t.Fatalf("want integer 42, got type %d value %d", got.GetType(), got.ToInteger())
	}

	mustNoErr(t, db.SetApplicationAttribute("app1", "val", lingo.NewLString("updated")))
	got, err = db.GetApplicationAttribute("app1", "val")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtString {
		t.Fatalf("want string after overwrite, got type %d", got.GetType())
	}

	names, err := db.GetApplicationAttributeNames("app1")
	mustNoErr(t, err)
	if len(names) != 1 || names[0] != "val" {
		t.Fatalf("want [val], got %v", names)
	}

	mustNoErr(t, db.DeleteApplicationAttribute("app1", "val"))
	got, err = db.GetApplicationAttribute("app1", "val")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtVoid {
		t.Error("attribute should be gone after delete")
	}
}

func TestPG_PlayerAttribute_SetGet(t *testing.T) {
	db := newTestPG(t)
	mustNoErr(t, db.CreateApplication("app1"))
	mustNoErr(t, db.SetPlayerAttribute("app1", "u1", "score", lingo.NewLInteger(7)))

	got, err := db.GetPlayerAttribute("app1", "u1", "score")
	mustNoErr(t, err)
	if got.ToInteger() != 7 {
		t.Fatalf("want 7, got %d", got.ToInteger())
	}
}

func TestPG_ApplicationAttribute_UnknownApp(t *testing.T) {
	db := newTestPG(t)
	if err := db.SetApplicationAttribute("nope", "k", lingo.NewLInteger(1)); err == nil {
		t.Error("expected error setting attribute on unknown application")
	}
}

// --- users ---

func TestPG_User_Lifecycle(t *testing.T) {
	db := newTestPG(t)
	mustNoErr(t, db.CreateUser("alice", "hash", 20))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)
	if u.Username != "alice" || u.UserLevel != 20 || u.UUID == "" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if u.CreatedAt.IsZero() {
		t.Error("created_at should be populated by the now() default")
	}

	mustNoErr(t, db.UpdateUserLevel("alice", 80))
	mustNoErr(t, db.UpdateUserPassword("alice", "newhash"))
	u, err = db.GetUser("alice")
	mustNoErr(t, err)
	if u.UserLevel != 80 || u.PasswordHash != "newhash" {
		t.Fatalf("update not applied: %+v", u)
	}

	mustNoErr(t, db.DeleteUser("alice"))
	if _, err := db.GetUser("alice"); !errors.Is(err, ports.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestPG_User_NotFoundErrors(t *testing.T) {
	db := newTestPG(t)
	if err := db.UpdateUserLevel("ghost", 1); !errors.Is(err, ports.ErrUserNotFound) {
		t.Errorf("UpdateUserLevel: want ErrUserNotFound, got %v", err)
	}
	if err := db.DeleteUser("ghost"); !errors.Is(err, ports.ErrUserNotFound) {
		t.Errorf("DeleteUser: want ErrUserNotFound, got %v", err)
	}
}

// --- bans ---

func TestPG_Ban_ByUserAndRevoke(t *testing.T) {
	db := newTestPG(t)
	mustNoErr(t, db.CreateUser("bob", "hash", 20))
	u, err := db.GetUser("bob")
	mustNoErr(t, err)

	mustNoErr(t, db.CreateBan(&u.ID, nil, "cheating", nil))
	ban, err := db.GetActiveBanByUserID(u.ID)
	mustNoErr(t, err)
	if ban.Reason != "cheating" || ban.UserID == nil || *ban.UserID != u.ID {
		t.Fatalf("unexpected ban: %+v", ban)
	}

	mustNoErr(t, db.RevokeBan(ban.ID))
	if _, err := db.GetActiveBanByUserID(u.ID); !errors.Is(err, ports.ErrBanNotFound) {
		t.Fatalf("want ErrBanNotFound after revoke, got %v", err)
	}
}

func TestPG_Ban_ByIP_Expired(t *testing.T) {
	db := newTestPG(t)
	ip := "10.0.0.1"

	past := time.Now().Add(-time.Hour)
	mustNoErr(t, db.CreateBan(nil, &ip, "temp", &past))
	if _, err := db.GetActiveBanByIP(ip); !errors.Is(err, ports.ErrBanNotFound) {
		t.Fatalf("expired ban should not be active, got %v", err)
	}

	future := time.Now().Add(time.Hour)
	mustNoErr(t, db.CreateBan(nil, &ip, "active", &future))
	ban, err := db.GetActiveBanByIP(ip)
	mustNoErr(t, err)
	if ban.IPAddress == nil || *ban.IPAddress != ip {
		t.Fatalf("unexpected ban: %+v", ban)
	}
}

// --- schema + query builder ---

func TestPG_Schema_And_QueryBuilder(t *testing.T) {
	db := newTestPG(t)
	qb := db.QueryBuilder()

	mustNoErr(t, db.CreateApplication("a1"))
	mustNoErr(t, db.CreateApplication("a2"))

	count, err := qb.Table("applications").Count()
	mustNoErr(t, err)
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	row, err := qb.Table("applications").Where("name", "a1").First()
	mustNoErr(t, err)
	if row == nil || row["name"] != "a1" {
		t.Fatalf("first = %v, want name a1", row)
	}

	mustNoErr(t, db.CreateUser("carol", "hash", 20))
	affected, err := qb.Table("users").Where("username", "carol").Update(map[string]interface{}{"user_level": 80})
	mustNoErr(t, err)
	if affected != 1 {
		t.Fatalf("update affected = %d, want 1", affected)
	}
	row, err = qb.Table("users").Where("username", "carol").Where("user_level", 80).First()
	mustNoErr(t, err)
	if row == nil {
		t.Fatal("expected chained-where row after update")
	}
	if level, ok := row["user_level"].(int64); !ok || level != 80 {
		t.Fatalf("user_level = %v (%T), want int64 80", row["user_level"], row["user_level"])
	}

	deleted, err := qb.Table("applications").Where("name", "a2").Delete()
	mustNoErr(t, err)
	if deleted != 1 {
		t.Fatalf("delete affected = %d, want 1", deleted)
	}

	if _, err := qb.Table("bad name!").Count(); err == nil {
		t.Error("expected error for invalid table name")
	}
}

// --- transactions ---

func createPGTxTable(t *testing.T, db *outbound.PostgresDB) {
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

func TestPG_Tx_CommitPersists(t *testing.T) {
	db := newTestPG(t)
	createPGTxTable(t, db)
	qb := db.QueryBuilder()

	txqb, ok := qb.(ports.TransactionalQueryBuilder)
	if !ok {
		t.Fatal("postgres query builder should be transactional")
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
	n, err := qb.Table("tx_test").Count()
	mustNoErr(t, err)
	if n != 1 {
		t.Fatalf("after commit want 1 row, got %d", n)
	}
}

func TestPG_Tx_RollbackDiscards(t *testing.T) {
	db := newTestPG(t)
	createPGTxTable(t, db)
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
	n, err := qb.Table("tx_test").Count()
	mustNoErr(t, err)
	if n != 0 {
		t.Fatalf("after rollback want 0 rows, got %d", n)
	}
}
