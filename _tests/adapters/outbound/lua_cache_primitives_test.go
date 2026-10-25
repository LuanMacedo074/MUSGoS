package outbound_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func runCachePrimitiveScript(t *testing.T, cache ports.Cache, body string) (*ports.ScriptResult, error) {
	t.Helper()
	dir := t.TempDir()
	mustNoErr(t, os.WriteFile(filepath.Join(dir, "primitive.lua"), []byte(body), 0600))
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 1, nil, nil, nil, nil, nil, cache, nil)
	return engine.Execute(&ports.ScriptMessage{Subject: "primitive"})
}

func TestLuaScriptEngine_CacheIncrementReturnsNewValue(t *testing.T) {
	cache := outbound.NewMemoryCache()
	t.Cleanup(func() { cache.Close() })
	_, err := runCachePrimitiveScript(t, cache, `
assert(mus.cache.increment("revision") == 1)
assert(mus.cache.increment("revision") == 2)
`)
	mustNoErr(t, err)
	value, err := cache.Get("revision")
	mustNoErr(t, err)
	if string(value) != "2" {
		t.Fatalf("revision = %q, want 2", value)
	}
}

func TestLuaScriptEngine_CacheIncrementErrorAbortsScript(t *testing.T) {
	cache := outbound.NewMemoryCache()
	t.Cleanup(func() { cache.Close() })
	mustNoErr(t, cache.Set("revision", []byte("not-an-integer"), 0))
	_, err := runCachePrimitiveScript(t, cache, `mus.cache.increment("revision")`)
	if err == nil || !strings.Contains(err.Error(), "cache increment") {
		t.Fatalf("expected cache increment error, got %v", err)
	}
}

func TestLuaScriptEngine_CacheSetAddOnceReportsFirstInsertion(t *testing.T) {
	cache := outbound.NewMemoryCache()
	t.Cleanup(func() { cache.Close() })
	_, err := runCachePrimitiveScript(t, cache, `
assert(mus.cache.setAddOnce("ledger", "action") == true)
assert(mus.cache.setAddOnce("ledger", "action") == false)
`)
	mustNoErr(t, err)
	present, err := cache.SetIsMember("ledger", "action")
	mustNoErr(t, err)
	if !present {
		t.Fatal("setAddOnce did not store the member")
	}
}
