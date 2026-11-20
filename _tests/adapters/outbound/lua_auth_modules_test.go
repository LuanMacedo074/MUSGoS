package outbound_test

import (
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	"golang.org/x/crypto/bcrypt"
)

func TestExecute_BcryptVerify_RightPassword(t *testing.T) {
	dir := setupScriptsDir(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	writeScript(t, dir, "verify", `
		mus.response(mus.bcrypt.verify("hunter2", mus.getContent()))
	`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "verify", SenderID: "u1", Content: lingo.NewLString(string(hash))})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := result.Content.(*lingo.LInteger)
	if !ok || b.Value != 1 {
		t.Errorf("verify(right) = %v, want truthy (LInteger 1)", result.Content)
	}
}

func TestExecute_BcryptVerify_WrongPassword(t *testing.T) {
	dir := setupScriptsDir(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	writeScript(t, dir, "verify", `
		mus.response(mus.bcrypt.verify("wrong", mus.getContent()))
	`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "verify", SenderID: "u1", Content: lingo.NewLString(string(hash))})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// false round-trips as LInteger 0 (LuaToLValue maps false -> 0).
	if b, ok := result.Content.(*lingo.LInteger); !ok || b.Value != 0 {
		t.Errorf("verify(wrong) = %v, want falsy (LInteger 0)", result.Content)
	}
}

func TestExecute_TokenIssueAndValidate_Roundtrip(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "tok", `
		local token = mus.token.issue({ user_id = "Frodo", claims = { user_level = "55", account_id = "9" } })
		local p = mus.token.validate(token)
		mus.response(p.user_id .. ":" .. p.claims.user_level .. ":" .. p.claims.account_id)
	`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil).
		WithAuthStores(outbound.NewCacheTokenStore(outbound.NewMemoryCache(), time.Minute), nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "tok", SenderID: "u1", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := lingo.StringValue(result.Content)
	if got != "Frodo:55:9" {
		t.Errorf("roundtrip = %q, want Frodo:55:9", got)
	}
}

func TestExecute_TokenValidate_UnknownReturnsNil(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "tok", `
		local p = mus.token.validate("bogus")
		mus.response(p == nil)
	`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil).
		WithAuthStores(outbound.NewCacheTokenStore(outbound.NewMemoryCache(), time.Minute), nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "tok", SenderID: "u1", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b, ok := result.Content.(*lingo.LInteger); !ok || b.Value != 1 {
		t.Errorf("validate(unknown)==nil = %v, want truthy", result.Content)
	}
}

func TestExecute_TokenModuleAbsentWithoutTokenStore(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "tok", `mus.response(mus.token == nil)`)

	// No injected token store -> no mus.token module (issue/validate need a store).
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "tok", SenderID: "u1", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b, ok := result.Content.(*lingo.LInteger); !ok || b.Value != 1 {
		t.Errorf("mus.token==nil = %v, want truthy (module absent)", result.Content)
	}
}
