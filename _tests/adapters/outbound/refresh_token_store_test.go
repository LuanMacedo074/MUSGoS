package outbound_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// The engine ships the refresh store but not the accounts table it authenticates
// against; that schema belongs to the application. This fixture provides the
// minimal accounts columns the store reads, plus the refresh tables.
func newRefreshTestDB(t *testing.T) *outbound.SQLiteDB {
	t.Helper()
	db := newTestDB(t)
	schema := ports.MigrationSQL{SQLite: []string{
		`CREATE TABLE accounts (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, banned INTEGER NOT NULL DEFAULT 0, user_level INTEGER NOT NULL DEFAULT 20)`,
		`CREATE TABLE refresh_families (id TEXT PRIMARY KEY, account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE, device_label TEXT NOT NULL, password_hash TEXT NOT NULL, created_at BIGINT NOT NULL, last_used_at BIGINT NOT NULL, expires_at BIGINT NOT NULL, revoked INTEGER NOT NULL DEFAULT 0, revision BIGINT NOT NULL DEFAULT 0)`,
		`CREATE INDEX refresh_families_account ON refresh_families(account_id)`,
		`CREATE TABLE refresh_tokens (token_hash TEXT PRIMARY KEY, family_id TEXT NOT NULL REFERENCES refresh_families(id) ON DELETE CASCADE, used INTEGER NOT NULL DEFAULT 0)`,
		`CREATE INDEX refresh_tokens_family ON refresh_tokens(family_id)`,
	}}
	err := runMigrations(db, &funcMigration{name: "refresh_fixture", up: func(tx ports.DBAdapter) error {
		return tx.(ports.SQLMigrationExecutor).ExecuteMigrationSQL(schema)
	}})
	if err != nil {
		t.Fatalf("create refresh schema: %v", err)
	}
	return db
}

func seedRefreshAccount(t *testing.T, db *outbound.SQLiteDB, email string) int64 {
	t.Helper()
	if err := db.QueryBuilder().Table("accounts").Insert(map[string]interface{}{"email": email, "password_hash": "hash"}); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	row, err := db.QueryBuilder().Table("accounts").Where("email", email).First()
	if err != nil || row == nil {
		t.Fatalf("read back account: %v", err)
	}
	return row["id"].(int64)
}

func TestRefreshTokenStorePersistsOnlySHA256Digest(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "digest@example.com")
	token, err := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil).Issue(id, "desktop")
	if err != nil || len(token) != 64 {
		t.Fatalf("want 256-bit opaque refresh token: %q %v", token, err)
	}
	rows, err := db.QueryBuilder().Table("refresh_tokens").Get()
	if err != nil || len(rows) != 1 {
		t.Fatalf("stored hashes: %v", err)
	}
	digest := sha256.Sum256([]byte(token))
	if rows[0]["token_hash"] != hex.EncodeToString(digest[:]) {
		t.Fatal("stored value is not SHA-256 digest")
	}
}

func TestRefreshTokenStoreRotatesAndReplayRevokesOnlyThatDevice(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "rotate@example.com")
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil)
	first, err := store.Issue(id, "desktop")
	mustNoErr(t, err)
	otherDevice, err := store.Issue(id, "laptop")
	mustNoErr(t, err)
	account, rotated, err := store.Rotate(first)
	if err != nil || account != id || rotated == "" || rotated == first {
		t.Fatalf("refresh must rotate: account=%d err=%v", account, err)
	}
	if account, _, _ = store.Rotate(first); account != 0 {
		t.Fatal("replay accepted")
	}
	if account, _, _ = store.Rotate(rotated); account != 0 {
		t.Fatal("replay did not revoke family")
	}
	if account, _, _ = store.Rotate(otherDevice); account != id {
		t.Fatal("replay revoked another device")
	}
}

func TestRefreshTokenStoreRejectsBannedAccountAndChangedPassword(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "policy@example.com")
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil)
	banned, err := store.Issue(id, "desktop")
	mustNoErr(t, err)
	_, err = db.QueryBuilder().Table("accounts").Where("id", id).Update(map[string]interface{}{"banned": 1})
	mustNoErr(t, err)
	if account, _, err := store.Rotate(banned); err != nil || account != 0 {
		t.Fatal("banned account refreshed")
	}
	_, err = db.QueryBuilder().Table("accounts").Where("id", id).Update(map[string]interface{}{"banned": 0})
	mustNoErr(t, err)
	reset, err := store.Issue(id, "desktop")
	mustNoErr(t, err)
	_, err = db.QueryBuilder().Table("accounts").Where("id", id).Update(map[string]interface{}{"password_hash": "replacement"})
	mustNoErr(t, err)
	if account, _, err := store.Rotate(reset); err != nil || account != 0 {
		t.Fatal("token outlived its family's password binding")
	}
	if account, _, err := store.Rotate(strings.Repeat("0", 64)); err != nil || account != 0 {
		t.Fatal("unknown token authenticated")
	}
}

func TestRefreshTokenStoreExpirySlidesThirtyDaysFromLastUse(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "expiry@example.com")
	now := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), func() time.Time { return now })
	token, err := store.Issue(id, "desktop")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(29 * 24 * time.Hour)
	account, rotated, err := store.Rotate(token)
	if err != nil || account != id || rotated == "" {
		t.Fatalf("valid token rejected: %v", err)
	}
	now = now.Add(29 * 24 * time.Hour)
	account, rotated, err = store.Rotate(rotated)
	if err != nil || account != id {
		t.Fatal("expiry did not slide")
	}
	now = now.Add(30 * 24 * time.Hour)
	account, _, err = store.Rotate(rotated)
	if err != nil || account != 0 {
		t.Fatal("expired token accepted at exact deadline")
	}
}

func TestRefreshTokenStoreConcurrentReplayRevokesFamily(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "race@example.com")
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil)
	token, err := store.Issue(id, "desktop")
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		account int64
		token   string
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			account, rotated, err := store.Rotate(token)
			results <- result{account, rotated, err}
		}()
	}
	close(start)
	var rotated string
	successes := 0
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err != nil {
			t.Fatalf("concurrent rotation must authenticate or reject without losing revocation: %v", result.err)
		}
		if result.account != 0 {
			successes++
			rotated = result.token
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent rotations succeeded %d times", successes)
	}
	account, _, err := store.Rotate(rotated)
	if err != nil || account != 0 {
		t.Fatal("concurrent replay did not revoke the rotated token")
	}
}

func TestRefreshTokenStoreDeviceLabelPreservesUTF8AtByteLimit(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "label@example.com")
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil)
	_, err := store.Issue(id, strings.Repeat("a", 127)+"é-desktop")
	if err != nil {
		t.Fatal(err)
	}
	family, err := db.QueryBuilder().Table("refresh_families").Where("account_id", id).First()
	if err != nil {
		t.Fatal(err)
	}
	if family["device_label"] != strings.Repeat("a", 127) {
		t.Fatal("device label must end on a complete UTF-8 rune within 128 bytes")
	}
}

// Hold a real database writer briefly; the store operation must survive the
// temporary contention without bypassing its transaction or losing writes.
func holdRefreshWriter(t *testing.T, db *outbound.SQLiteDB, accountID int64) {
	t.Helper()
	tx, err := db.QueryBuilder().(ports.TransactionalQueryBuilder).Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Table("accounts").Where("id", accountID).Increment("user_level", 0); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	released := make(chan error, 1)
	time.AfterFunc(50*time.Millisecond, func() { released <- tx.Rollback() })
	t.Cleanup(func() {
		if err := <-released; err != nil {
			t.Error(err)
		}
	})
}

func TestRefreshTokenStoreIssueSurvivesTemporaryDatabaseContention(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "issue-contention@example.com")
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil)
	holdRefreshWriter(t, db, id)
	token, err := store.Issue(id, "desktop")
	if err != nil {
		t.Fatalf("issue failed during temporary contention: %v", err)
	}
	account, _, err := store.Rotate(token)
	if err != nil || account != id {
		t.Fatal("issued token does not authenticate its account")
	}
}

func TestRefreshTokenStoreRevokeSurvivesTemporaryDatabaseContention(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "revoke-contention@example.com")
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil)
	token, err := store.Issue(id, "desktop")
	if err != nil {
		t.Fatal(err)
	}
	holdRefreshWriter(t, db, id)
	if err := store.Revoke(token); err != nil {
		t.Fatalf("revoke failed during temporary contention: %v", err)
	}
	account, _, err := store.Rotate(token)
	if err != nil || account != 0 {
		t.Fatal("revoked token still authenticates")
	}
}

// A commit failure is a database fault; keep all real queries enrolled so the
// store's rollback is observable afterwards.
type failingCommitBuilder struct {
	ports.TransactionalQueryBuilder
}

func (b failingCommitBuilder) Begin() (ports.Tx, error) {
	tx, err := b.TransactionalQueryBuilder.Begin()
	if err != nil {
		return nil, err
	}
	return failingCommitTx{Tx: tx}, nil
}

type failingCommitTx struct{ ports.Tx }

func (tx failingCommitTx) Commit() error { return fmt.Errorf("storage unavailable during commit") }

func TestRefreshTokenStoreFailedRevocationCommitLeavesDeviceUsable(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "revoke-rollback@example.com")
	store := outbound.NewRefreshTokenStore(db.QueryBuilder(), nil)
	token, err := store.Issue(id, "desktop")
	if err != nil {
		t.Fatal(err)
	}
	failing := outbound.NewRefreshTokenStore(failingCommitBuilder{
		TransactionalQueryBuilder: db.QueryBuilder().(ports.TransactionalQueryBuilder),
	}, nil)
	if err := failing.Revoke(token); err == nil {
		t.Fatal("revocation succeeded despite a failed storage commit")
	}
	account, rotated, err := store.Rotate(token)
	if err != nil || account != id || rotated == "" {
		t.Fatalf("failed revocation changed the remembered device: account=%d err=%v", account, err)
	}
	if err := store.Revoke(rotated); err != nil {
		t.Fatal(err)
	}
	account, _, err = store.Rotate(rotated)
	if err != nil || account != 0 {
		t.Fatalf("revocation retry did not disable the device: account=%d err=%v", account, err)
	}
}

func TestLuaRefreshModuleIssuesRotatesAndRevokes(t *testing.T) {
	db := newRefreshTestDB(t)
	id := seedRefreshAccount(t, db, "lua@example.com")
	dir := t.TempDir()
	writeScript(t, dir, "refresh", fmt.Sprintf(`
local first = mus.refresh.issue(%d, "desktop")
local account, rotated = mus.refresh.rotate(first)
local replayed = mus.refresh.rotate(first)
local second = mus.refresh.issue(%d, "laptop")
mus.refresh.revoke(second)
local revoked = mus.refresh.rotate(second)
mus.response(string.format("%%d:%%d:%%s:%%s", #first, account, tostring(replayed), tostring(revoked)))
`, id, id))
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, db, db.QueryBuilder(), nil, nil, nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "refresh", SenderID: "sys", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := fmt.Sprintf("64:%d:nil:nil", id)
	if got := lingo.StringValue(result.Content); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
