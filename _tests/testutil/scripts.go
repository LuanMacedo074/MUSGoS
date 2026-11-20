package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// NewScriptsDir creates an isolated directory for authored script probes.
func NewScriptsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// WriteScript writes a named Lua probe in dir.
func WriteScript(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name+".lua"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
}

// WriteScriptNested writes a named Lua probe in a subdirectory.
func WriteScriptNested(t *testing.T, dir, sub, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", sub, err)
	}
	if err := os.WriteFile(filepath.Join(dir, sub, name+".lua"), []byte(content), 0o644); err != nil {
		t.Fatalf("write nested script: %v", err)
	}
}
