package outbound_test

import (
	"os"
	"path/filepath"
	"testing"

	"fsos-server/external/migrations"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/services"
	"fsos-server/internal/domain/types/lingo"
)

func newTestDB(t *testing.T) *outbound.SQLiteDB {
	t.Helper()
	dir := t.TempDir()
	db, err := outbound.NewSQLiteDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	runner := services.NewMigrationRunner(db, db, migrations.All)
	if err := runner.RunPending(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSQLiteDB_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := outbound.NewSQLiteDB(dbPath)
	mustNoErr(t, err)
	defer db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file should have been created")
	}
}

func TestNewSQLiteDB_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "subdir", "test.db")

	db, err := outbound.NewSQLiteDB(dbPath)
	mustNoErr(t, err)
	defer db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file should have been created in subdirectory")
	}
}

func TestNewSQLiteDB_InvalidPath(t *testing.T) {
	_, err := outbound.NewSQLiteDB("/nonexistent/dir/test.db")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

// --- DBAdmin ---

func TestCreateApplication(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("testApp"))
}

func TestCreateApplication_Duplicate(t *testing.T) {
	db := newTestDB(t)

	mustNoErr(t, db.CreateApplication("testApp"))
	err := db.CreateApplication("testApp")
	if err == nil {
		t.Error("expected error for duplicate application")
	}
}

func TestDeleteApplication(t *testing.T) {
	db := newTestDB(t)

	mustNoErr(t, db.CreateApplication("testApp"))
	mustNoErr(t, db.DeleteApplication("testApp"))

	// Should be able to create again after delete
	mustNoErr(t, db.CreateApplication("testApp"))
}

func TestDeleteApplication_NonExistent(t *testing.T) {
	db := newTestDB(t)

	err := db.DeleteApplication("noapp")
	if err == nil {
		t.Error("expected error when deleting non-existent application")
	}
}

func TestDeleteApplication_CascadesAttributes(t *testing.T) {
	db := newTestDB(t)

	mustNoErr(t, db.CreateApplication("app1"))
	mustNoErr(t, db.SetApplicationAttribute("app1", "key", lingo.NewLInteger(1)))
	mustNoErr(t, db.SetPlayerAttribute("app1", "user1", "score", lingo.NewLInteger(100)))
	mustNoErr(t, db.DeleteApplication("app1"))

	// Recreate and verify attributes are gone
	mustNoErr(t, db.CreateApplication("app1"))

	got, err := db.GetApplicationAttribute("app1", "key")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtVoid {
		t.Error("application attributes should have been cascaded on delete")
	}

	got2, err := db.GetPlayerAttribute("app1", "user1", "score")
	mustNoErr(t, err)
	if got2.GetType() != lingo.VtVoid {
		t.Error("player attributes should have been cascaded on delete")
	}
}

// --- DBApplication attributes ---

func TestApplicationAttribute_SetGet_Integer(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "score", lingo.NewLInteger(42)))

	got, err := db.GetApplicationAttribute("app1", "score")
	mustNoErr(t, err)

	if got.GetType() != lingo.VtInteger {
		t.Errorf("expected integer type, got %d", got.GetType())
	}
	if got.ToInteger() != 42 {
		t.Errorf("expected 42, got %d", got.ToInteger())
	}
}

func TestApplicationAttribute_SetGet_Float(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "ratio", lingo.NewLFloat(3.14)))

	got, err := db.GetApplicationAttribute("app1", "ratio")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtFloat {
		t.Errorf("expected float type, got %d", got.GetType())
	}
	if got.ToDouble() != 3.14 {
		t.Errorf("expected 3.14, got %f", got.ToDouble())
	}
}

func TestApplicationAttribute_SetGet_String(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "greeting", lingo.NewLString("hello world")))

	got, err := db.GetApplicationAttribute("app1", "greeting")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtString {
		t.Errorf("expected string type, got %d", got.GetType())
	}
	ls, ok := got.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", got)
	}
	if ls.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", ls.Value)
	}
}

func TestApplicationAttribute_SetGet_Symbol(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "sym", lingo.NewLSymbol("mySymbol")))

	got, err := db.GetApplicationAttribute("app1", "sym")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtSymbol {
		t.Errorf("expected symbol type, got %d", got.GetType())
	}
	ls, ok := got.(*lingo.LSymbol)
	if !ok {
		t.Fatalf("expected *LSymbol, got %T", got)
	}
	if ls.Value != "mySymbol" {
		t.Errorf("expected 'mySymbol', got %q", ls.Value)
	}
}

func TestApplicationAttribute_Overwrite(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "val", lingo.NewLInteger(1)))
	mustNoErr(t, db.SetApplicationAttribute("app1", "val", lingo.NewLString("updated")))

	got, err := db.GetApplicationAttribute("app1", "val")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtString {
		t.Errorf("expected string after overwrite, got type %d", got.GetType())
	}
	ls, ok := got.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", got)
	}
	if ls.Value != "updated" {
		t.Errorf("expected 'updated', got %q", ls.Value)
	}
}

func TestApplicationAttribute_GetNonExistent(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	got, err := db.GetApplicationAttribute("app1", "missing")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtVoid {
		t.Errorf("expected void for missing attribute, got type %d", got.GetType())
	}
}

func TestApplicationAttribute_GetNames(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "a", lingo.NewLInteger(1)))
	mustNoErr(t, db.SetApplicationAttribute("app1", "b", lingo.NewLString("x")))

	names, err := db.GetApplicationAttributeNames("app1")
	mustNoErr(t, err)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestApplicationAttribute_Delete(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetApplicationAttribute("app1", "key", lingo.NewLInteger(1)))
	mustNoErr(t, db.DeleteApplicationAttribute("app1", "key"))

	got, err := db.GetApplicationAttribute("app1", "key")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtVoid {
		t.Errorf("expected void after delete, got type %d", got.GetType())
	}
}

func TestApplicationAttribute_NonExistentApp(t *testing.T) {
	db := newTestDB(t)

	err := db.SetApplicationAttribute("noapp", "key", lingo.NewLInteger(1))
	if err == nil {
		t.Error("expected error for non-existent application")
	}
}

// --- DBPlayer attributes ---

func TestPlayerAttribute_SetGetDelete(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateApplication("app1"))

	mustNoErr(t, db.SetPlayerAttribute("app1", "user1", "score", lingo.NewLInteger(100)))

	got, err := db.GetPlayerAttribute("app1", "user1", "score")
	mustNoErr(t, err)
	if got.ToInteger() != 100 {
		t.Errorf("expected 100, got %d", got.ToInteger())
	}

	// Different user, same attr name
	got2, err := db.GetPlayerAttribute("app1", "user2", "score")
	mustNoErr(t, err)
	if got2.GetType() != lingo.VtVoid {
		t.Error("different user should not have the attribute")
	}

	// GetNames
	mustNoErr(t, db.SetPlayerAttribute("app1", "user1", "level", lingo.NewLInteger(5)))
	names, err := db.GetPlayerAttributeNames("app1", "user1")
	mustNoErr(t, err)
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}

	// Delete
	mustNoErr(t, db.DeletePlayerAttribute("app1", "user1", "score"))
	got3, err := db.GetPlayerAttribute("app1", "user1", "score")
	mustNoErr(t, err)
	if got3.GetType() != lingo.VtVoid {
		t.Error("expected void after delete")
	}
}

// --- DBUser attributes ---

func TestUserAttribute_SetGetDelete(t *testing.T) {
	db := newTestDB(t)

	mustNoErr(t, db.SetUserAttribute("client1", "token", lingo.NewLString("session-data")))

	got, err := db.GetUserAttribute("client1", "token")
	mustNoErr(t, err)
	ls, ok := got.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", got)
	}
	if ls.Value != "session-data" {
		t.Errorf("expected 'session-data', got %q", ls.Value)
	}

	// Different client
	got2, err := db.GetUserAttribute("client2", "token")
	mustNoErr(t, err)
	if got2.GetType() != lingo.VtVoid {
		t.Error("different client should not have the attribute")
	}

	// GetNames
	mustNoErr(t, db.SetUserAttribute("client1", "flag", lingo.NewLInteger(1)))
	names, err := db.GetUserAttributeNames("client1")
	mustNoErr(t, err)
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}

	// Delete
	mustNoErr(t, db.DeleteUserAttribute("client1", "token"))
	got3, err := db.GetUserAttribute("client1", "token")
	mustNoErr(t, err)
	if got3.GetType() != lingo.VtVoid {
		t.Error("expected void after delete")
	}
}

// --- MigrationTracker ---

func TestMigrationTracker_EmptyInitially(t *testing.T) {
	dir := t.TempDir()
	db, err := outbound.NewSQLiteDB(filepath.Join(dir, "test.db"))
	mustNoErr(t, err)
	defer db.Close()

	applied, err := db.GetAppliedMigrations()
	mustNoErr(t, err)
	if len(applied) != 0 {
		t.Errorf("expected 0 applied migrations, got %d", len(applied))
	}
}

func TestMigrationTracker_MarkAndGet(t *testing.T) {
	dir := t.TempDir()
	db, err := outbound.NewSQLiteDB(filepath.Join(dir, "test.db"))
	mustNoErr(t, err)
	defer db.Close()

	mustNoErr(t, db.MarkMigrationApplied("20260101000000_first"))
	mustNoErr(t, db.MarkMigrationApplied("20260102000000_second"))

	applied, err := db.GetAppliedMigrations()
	mustNoErr(t, err)
	if len(applied) != 2 {
		t.Fatalf("expected 2 applied, got %d", len(applied))
	}
	if applied[0] != "20260101000000_first" {
		t.Errorf("expected first migration name, got %q", applied[0])
	}
	if applied[1] != "20260102000000_second" {
		t.Errorf("expected second migration name, got %q", applied[1])
	}
}

func TestMigrationTracker_DuplicateFails(t *testing.T) {
	dir := t.TempDir()
	db, err := outbound.NewSQLiteDB(filepath.Join(dir, "test.db"))
	mustNoErr(t, err)
	defer db.Close()

	mustNoErr(t, db.MarkMigrationApplied("20260101000000_first"))
	err = db.MarkMigrationApplied("20260101000000_first")
	if err == nil {
		t.Error("expected error for duplicate migration mark")
	}
}
