package outbound_test

import (
	"os"
	"path/filepath"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// writeScriptNested writes dir/sub/name.lua, creating the subdirectory.
func writeScriptNested(t *testing.T, dir, sub, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", sub, err)
	}
	if err := os.WriteFile(filepath.Join(dir, sub, name+".lua"), []byte(content), 0o644); err != nil {
		t.Fatalf("write nested script: %v", err)
	}
}

// Subjects come from the network; the engine must never resolve one that
// escapes the scripts directory.
func TestPathSec_HasScriptRejectsTraversal(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "safe", `mus.response(1)`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

	if !engine.HasScript("safe") {
		t.Fatal("expected HasScript(safe) to be true")
	}

	bad := []string{
		"../secret",
		"../../etc/passwd",
		"..",
		"foo/../../bar",
		"/etc/passwd",
		"",
	}
	for _, subj := range bad {
		if engine.HasScript(subj) {
			t.Errorf("HasScript(%q) must be false (path traversal / escape)", subj)
		}
	}
}

func TestPathSec_ExecuteRejectsTraversal(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "safe", `mus.response(1)`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

	for _, subj := range []string{"../secret", "../../etc/passwd", "/etc/passwd", ".."} {
		_, err := engine.Execute(&ports.ScriptMessage{Subject: subj, SenderID: "u", Content: lingo.NewLVoid()})
		if err == nil {
			t.Errorf("Execute(%q) must return an error", subj)
		}
	}

	// A legitimate nested subject still works.
	writeScriptNested(t, dir, "users", "hello", `mus.response(7)`)
	res, err := engine.Execute(&ports.ScriptMessage{Subject: "users/hello", SenderID: "u", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("nested subject should execute: %v", err)
	}
	n, ok := res.Content.(*lingo.LInteger)
	if !ok || n.Value != 7 {
		t.Fatalf("expected 7 from users/hello, got %v", res.Content)
	}
}
