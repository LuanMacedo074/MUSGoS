package outbound_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fsos-server/external/migrations"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
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

// --- DBUser ---

func TestCreateUser(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))
}

func TestCreateUser_Duplicate(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))
	err := db.CreateUser("alice", "hash456", "salt456", 20)
	if err == nil {
		t.Error("expected error for duplicate username")
	}
}

func TestGetUser(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)

	if u.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", u.Username)
	}
	if u.PasswordHash != "hash123" {
		t.Errorf("expected password_hash 'hash123', got %q", u.PasswordHash)
	}
	if u.Salt != "salt123" {
		t.Errorf("expected salt 'salt123', got %q", u.Salt)
	}
	if u.UserLevel != 20 {
		t.Errorf("expected user_level 20, got %d", u.UserLevel)
	}
	if u.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestGetUser_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetUser("nobody")
	if !errors.Is(err, ports.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))
	mustNoErr(t, db.DeleteUser("alice"))

	_, err := db.GetUser("alice")
	if !errors.Is(err, ports.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestDeleteUser_CascadesBans(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)

	mustNoErr(t, db.CreateBan(&u.ID, nil, "bad behavior", nil))

	ban, err := db.GetActiveBanByUserID(u.ID)
	mustNoErr(t, err)
	if ban == nil {
		t.Fatal("expected ban to exist before delete")
	}

	mustNoErr(t, db.DeleteUser("alice"))

	_, err = db.GetActiveBanByUserID(u.ID)
	if !errors.Is(err, ports.ErrBanNotFound) {
		t.Errorf("expected ErrBanNotFound after cascade delete, got %v", err)
	}
}

func TestUpdateUserLevel(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))
	mustNoErr(t, db.UpdateUserLevel("alice", 80))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)
	if u.UserLevel != 80 {
		t.Errorf("expected user_level 80, got %d", u.UserLevel)
	}
}

func TestUpdateUserPassword(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))
	mustNoErr(t, db.UpdateUserPassword("alice", "newhash", "newsalt"))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)
	if u.PasswordHash != "newhash" {
		t.Errorf("expected password_hash 'newhash', got %q", u.PasswordHash)
	}
	if u.Salt != "newsalt" {
		t.Errorf("expected salt 'newsalt', got %q", u.Salt)
	}
}

// --- DBBan ---

func TestCreateBan_ByUserID(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)

	mustNoErr(t, db.CreateBan(&u.ID, nil, "spamming", nil))

	ban, err := db.GetActiveBanByUserID(u.ID)
	mustNoErr(t, err)
	if ban == nil {
		t.Fatal("expected ban to exist")
	}
	if *ban.UserID != u.ID {
		t.Errorf("expected user_id %d, got %d", u.ID, *ban.UserID)
	}
	if ban.Reason != "spamming" {
		t.Errorf("expected reason 'spamming', got %q", ban.Reason)
	}
}

func TestCreateBan_NeitherUserNorIP(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateBan(nil, nil, "orphan ban", nil)
	if err == nil {
		t.Error("expected error when both user_id and ip_address are nil")
	}
}

func TestCreateBan_ByIP(t *testing.T) {
	db := newTestDB(t)
	ip := "192.168.1.100"
	mustNoErr(t, db.CreateBan(nil, &ip, "abuse", nil))

	ban, err := db.GetActiveBanByIP(ip)
	mustNoErr(t, err)
	if ban == nil {
		t.Fatal("expected ban to exist")
	}
	if *ban.IPAddress != ip {
		t.Errorf("expected ip_address %q, got %q", ip, *ban.IPAddress)
	}
}

func TestGetActiveBan_Expired(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)

	past := time.Now().Add(-1 * time.Hour)
	mustNoErr(t, db.CreateBan(&u.ID, nil, "temp ban", &past))

	_, err = db.GetActiveBanByUserID(u.ID)
	if !errors.Is(err, ports.ErrBanNotFound) {
		t.Errorf("expected ErrBanNotFound for expired ban, got %v", err)
	}
}

func TestGetActiveBan_Permanent(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)

	mustNoErr(t, db.CreateBan(&u.ID, nil, "permanent", nil))

	ban, err := db.GetActiveBanByUserID(u.ID)
	mustNoErr(t, err)
	if ban == nil {
		t.Fatal("expected permanent ban to be returned")
	}
	if ban.ExpiresAt != nil {
		t.Error("expected nil expires_at for permanent ban")
	}
}

func TestRevokeBan(t *testing.T) {
	db := newTestDB(t)
	mustNoErr(t, db.CreateUser("alice", "hash123", "salt123", 20))

	u, err := db.GetUser("alice")
	mustNoErr(t, err)

	mustNoErr(t, db.CreateBan(&u.ID, nil, "revokable", nil))

	ban, err := db.GetActiveBanByUserID(u.ID)
	mustNoErr(t, err)
	if ban == nil {
		t.Fatal("expected ban to exist")
	}

	mustNoErr(t, db.RevokeBan(ban.ID))

	_, err = db.GetActiveBanByUserID(u.ID)
	if !errors.Is(err, ports.ErrBanNotFound) {
		t.Errorf("expected ErrBanNotFound after revoke, got %v", err)
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
