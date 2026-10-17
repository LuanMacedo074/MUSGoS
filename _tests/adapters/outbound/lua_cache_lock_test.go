package outbound_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestLuaScriptEngine_ScopedLockRejectsNewLocksInsideCallback(t *testing.T) {
	for _, acquire := range []string{`mus.cache.lock("other")`, `mus.cache.withLock("other", function() end)`} {
		t.Run(acquire, func(t *testing.T) {
			dir := t.TempDir()
			mustNoErr(t, os.WriteFile(filepath.Join(dir, "operation.lua"), []byte(`
mus.cache.lock("existing")
mus.cache.withLock("scope", function()
 mus.cache.lock("existing")
 mus.cache.withLock("scope", function() end)
 `+acquire+`
end)
`), 0600))
			cache := outbound.NewMemoryCache()
			t.Cleanup(func() { cache.Close() })
			engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 1, nil, nil, nil, nil, nil, cache, nil)
			_, err := engine.Execute(&ports.ScriptMessage{Subject: "operation"})
			if err == nil || !strings.Contains(err.Error(), "cannot acquire another lock inside a scoped lock") {
				t.Fatalf("expected scoped lock ordering error, got %v", err)
			}
		})
	}
}

func TestLuaScriptEngine_ScopedLockReturnsValuesAndReleasesBeforeScriptExit(t *testing.T) {
	for _, prefix := range []string{"", `
local ok = pcall(function()
 mus.cache.withLock("operation", function() error("failed callback") end)
end)
assert(not ok)
`} {
		t.Run(fmt.Sprintf("recovered-error=%t", prefix != ""), func(t *testing.T) {
			dir := t.TempDir()
			mustNoErr(t, os.WriteFile(filepath.Join(dir, "operation.lua"), []byte(prefix+`
local value, absent, number = mus.cache.withLock("operation", function()
 return "ready", nil, 7
end)
assert(value == "ready" and absent == nil and number == 7)
mus.cache.get("after-scope")
mus.response("finished")
`), 0600))
			cache := &scopedLockBarrier{Cache: outbound.NewMemoryCache(), after: make(chan struct{}), resume: make(chan struct{})}
			var once sync.Once
			resume := func() { once.Do(func() { close(cache.resume) }) }
			t.Cleanup(resume)
			t.Cleanup(func() { cache.Close() })
			engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, cache, nil)
			done := make(chan error, 1)
			go func() {
				_, err := engine.Execute(&ports.ScriptMessage{Subject: "operation"})
				done <- err
			}()
			select {
			case <-cache.after:
			case err := <-done:
				t.Fatalf("script did not reach the end of the lock scope: %v", err)
			case <-time.After(3 * time.Second):
				t.Fatal("script did not reach the end of the lock scope")
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			lease, err := cache.AcquireLock(ctx, "operation")
			resume()
			mustNoErr(t, err)
			mustNoErr(t, lease.Release())
			mustNoErr(t, <-done)
		})
	}
}

type scopedLockBarrier struct {
	ports.Cache
	after, resume chan struct{}
}

func (c *scopedLockBarrier) Get(key string) ([]byte, error) {
	if key == "after-scope" {
		close(c.after)
		<-c.resume
	}
	return c.Cache.Get(key)
}

func TestLuaScriptEngine_OperationLocksReleaseOnEveryExit(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		for _, exit := range []string{"return", "error('failed operation')", "while true do end"} {
			t.Run(fmt.Sprintf("scoped=%t/%s", scoped, exit), func(t *testing.T) {
				dir := t.TempDir()
				script := "mus.cache.lock('operation')\nmus.cache.lock('operation')\n" + exit
				if scoped {
					script = "mus.cache.withLock('operation', function()\nmus.cache.withLock('operation', function() end)\n" + exit + "\nend)"
				}
				if err := os.WriteFile(filepath.Join(dir, "operation.lua"), []byte(script), 0600); err != nil {
					t.Fatal(err)
				}
				cache := outbound.NewMemoryCache()
				t.Cleanup(func() { cache.Close() })
				engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 1, nil, nil, nil, nil, nil, cache, nil)
				_, err := engine.Execute(&ports.ScriptMessage{Subject: "operation"})
				if (exit == "return") != (err == nil) {
					t.Fatalf("unexpected operation result: %v", err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				release, err := cache.AcquireLock(ctx, "operation")
				if err != nil {
					t.Fatalf("finished operation retained its lock: %v", err)
				}
				if err := release.Release(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
