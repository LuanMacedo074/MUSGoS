package outbound_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestLuaScriptEngine_OperationLocksReleaseOnEveryExit(t *testing.T) {
	for _, exit := range []string{"return", "error('failed operation')", "while true do end"} {
		t.Run(exit, func(t *testing.T) {
			dir := t.TempDir()
			script := "mus.cache.lock('operation')\nmus.cache.lock('operation')\n" + exit
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
