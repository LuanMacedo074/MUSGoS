package factory_test

import (
	"os"
	"path/filepath"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/factory"
)

func loaderType(t *testing.T, sandboxedDir string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "probe.lua"), []byte(`mus.response(type(mus.loadSandboxed))`), 0644); err != nil {
		t.Fatal(err)
	}
	engine := factory.NewScriptEngine(dir, sandboxedDir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "probe", SenderID: "u1", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	return lingo.StringValue(result.Content)
}

func TestNewScriptEngine_SandboxedDirEnablesTheLoader(t *testing.T) {
	if got := loaderType(t, "chunks"); got != "function" {
		t.Errorf("type(mus.loadSandboxed) = %q, want function", got)
	}
}

func TestNewScriptEngine_EmptySandboxedDirLeavesTheLoaderOff(t *testing.T) {
	if got := loaderType(t, ""); got != "nil" {
		t.Errorf("type(mus.loadSandboxed) = %q, want nil", got)
	}
}
