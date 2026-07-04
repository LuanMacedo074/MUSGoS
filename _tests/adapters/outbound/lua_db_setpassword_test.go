package outbound_test

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

func TestLuaDB_SetPassword(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "setpw", `
mus.db.createUser("hero", "oldpass", 20)
mus.db.setPassword("hero", "newpass")
`)

	db := newTestDB(t)
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, db, db.QueryBuilder(), nil, nil, nil)

	if _, err := engine.Execute(&ports.ScriptMessage{Subject: "setpw", SenderID: "sys", Content: lingo.NewLVoid()}); err != nil {
		t.Fatalf("execute: %v", err)
	}

	row, err := db.QueryBuilder().Table("users").Where("username", "hero").First()
	if err != nil {
		t.Fatalf("query user: %v", err)
	}
	if row == nil {
		t.Fatal("user hero not found")
	}
	hash, ok := row["password_hash"].(string)
	if !ok {
		t.Fatalf("password_hash not a string: %T", row["password_hash"])
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("newpass")); err != nil {
		t.Errorf("stored hash should validate against the new password: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("oldpass")); err == nil {
		t.Error("stored hash must no longer validate against the old password")
	}
}
