package factory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"musgos/_tests/testutil"
	"musgos/internal/adapters/outbound"
	"musgos/internal/domain/ports"
	"musgos/internal/domain/types/lingo"
	"musgos/internal/factory"
)

func TestAuthStoresLuaClaimsAndLegacyTokens(t *testing.T) {
	dir := t.TempDir()
	script := `
assert(mus.token.user_level_claim == "user_level")
local p = mus.token.validate("old-token")
assert(p.user_id == "reader" and p.claims.tenant == "blue" and p.claims.user_level == "40")
local token = mus.token.issue({user_id = p.user_id, claims = {tenant = p.claims.tenant, arbitrary = "olá", [mus.token.user_level_claim] = p.claims[mus.token.user_level_claim]}})
local fresh = mus.token.validate(token)
assert(fresh.user_id == "reader" and fresh.claims.tenant == "blue" and fresh.claims.arbitrary == "olá")
assert(mus.token.validate(token) == nil)
mus.response(fresh.claims.user_level)
`
	if err := os.WriteFile(filepath.Join(dir, "probe.lua"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	cache := testutil.NewMockCache()
	if err := cache.Set("authtoken:old-token", []byte(`{"login":"reader","tenant":"blue","user_level":40}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	stores := factory.NewAuthStores(cache, nil, factory.AuthConfig{LegacyUserIDClaim: "login"})
	engine := factory.NewScriptEngine(dir, "", &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, cache, nil, stores)
	res, err := engine.Execute(&ports.ScriptMessage{Subject: "probe"})
	if err != nil {
		t.Fatal(err)
	}
	if lingo.StringValue(res.Content) != "40" {
		t.Fatal("claims lost through Lua API")
	}
}

func TestLuaScriptEngineDoesNotConstructAuthFromCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "probe.lua"), []byte(`mus.response(type(mus.token) .. ":" .. type(mus.refresh))`), 0600); err != nil {
		t.Fatal(err)
	}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, testutil.NewMockCache(), nil)
	res, err := engine.Execute(&ports.ScriptMessage{Subject: "probe"})
	if err != nil {
		t.Fatal(err)
	}
	if got := lingo.StringValue(res.Content); got != "nil:nil" {
		t.Fatalf("unconfigured auth modules = %q", got)
	}
}
