package outbound_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// sandboxedEngine returns an engine whose sandboxed chunks live in
// <scripts>/chunks, plus the scripts dir for writing the probe and the chunks.
func sandboxedEngine(t *testing.T) (*outbound.LuaScriptEngine, string) {
	t.Helper()
	dir := setupScriptsDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "chunks"), 0755); err != nil {
		t.Fatal(err)
	}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil).
		WithSandboxedChunkDir("chunks")
	return engine, dir
}

func writeChunk(t *testing.T, dir, id, code string) {
	t.Helper()
	writeScript(t, filepath.Join(dir, "chunks"), id, code)
}

func probe(t *testing.T, engine *outbound.LuaScriptEngine, dir, code string) string {
	t.Helper()
	writeScript(t, dir, "probe", code)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "probe", SenderID: "u1", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	s, ok := result.Content.(*lingo.LString)
	if !ok {
		t.Fatalf("probe response = %T, want *LString", result.Content)
	}
	return s.Value
}

func TestLoadSandboxed_ReturnsTheChunkTable(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeChunk(t, dir, "greeter", `return { greet = function(name) return "hi " .. name end }`)

	got := probe(t, engine, dir, `local m = mus.loadSandboxed("greeter")
mus.response(m.greet("bob"))`)
	if got != "hi bob" {
		t.Errorf("response = %q, want %q", got, "hi bob")
	}
}

func TestLoadSandboxed_UsesTheConfiguredDirectory(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeScript(t, dir, "elsewhere", `return {}`)

	got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "elsewhere")
mus.response(tostring(ok))`)
	if got != "false" {
		t.Errorf("loading a file outside the configured directory: ok = %s, want false", got)
	}
}

func TestLoadSandboxed_AbsentWithoutAConfiguredDirectory(t *testing.T) {
	dir := setupScriptsDir(t)
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)

	if got := probe(t, engine, dir, `mus.response(type(mus.loadSandboxed))`); got != "nil" {
		t.Errorf("type(mus.loadSandboxed) = %q, want nil", got)
	}
}

func TestLoadSandboxed_RejectsInvalidIDs(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeScript(t, dir, "secret", `return {}`)
	for _, id := range []string{"../secret", "a/b", "a.b", ""} {
		t.Run(id, func(t *testing.T) {
			got := probe(t, engine, dir, fmt.Sprintf(`local ok, err = pcall(mus.loadSandboxed, %q)
mus.response(tostring(ok) .. "|" .. tostring(err))`, id))
			if !strings.HasPrefix(got, "false|") || !strings.Contains(got, "invalid sandboxed chunk id") {
				t.Errorf("response = %q, want invalid id error", got)
			}
		})
	}
}

func TestLoadSandboxed_RejectsADirectoryOutsideTheScriptsDir(t *testing.T) {
	dir := setupScriptsDir(t)
	outside := t.TempDir()
	writeScript(t, outside, "escape", `return {}`)
	rel, err := filepath.Rel(dir, outside)
	if err != nil {
		t.Fatal(err)
	}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil).
		WithSandboxedChunkDir(rel)

	got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "escape")
mus.response(tostring(ok) .. "|" .. tostring(err))`)
	if !strings.HasPrefix(got, "false|") || !strings.Contains(got, "outside the scripts directory") {
		t.Errorf("response = %q, want containment error", got)
	}
}

func TestLoadSandboxed_ChunkSeesOnlySafeGlobals(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeChunk(t, dir, "probe_env", `return {
  mus = type(mus), require = type(require), dofile = type(dofile),
  getfenv = type(getfenv), setfenv = type(setfenv), loadstring = type(loadstring),
  pairs = type(pairs), math = type(math), string = type(string), table = type(table),
}`)

	got := probe(t, engine, dir, `local e = mus.loadSandboxed("probe_env")
mus.response(table.concat({e.mus, e.require, e.dofile, e.getfenv, e.setfenv, e.loadstring,
  e.pairs, e.math, e.string, e.table}, ","))`)
	want := "nil,nil,nil,nil,nil,nil,function,table,table,table"
	if got != want {
		t.Errorf("chunk globals = %q, want %q", got, want)
	}
}

func TestLoadSandboxed_ChunkWritesDoNotLeakIntoTheCaller(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeChunk(t, dir, "leaky", `leaked = "yes"
string.upper = function() return "pwned" end
return {}`)

	got := probe(t, engine, dir, `mus.loadSandboxed("leaky")
mus.response(tostring(leaked) .. "|" .. string.upper("ok"))`)
	if got != "nil|OK" {
		t.Errorf("caller state after chunk = %q, want nil|OK", got)
	}
}

func TestLoadSandboxed_NonTableReturnIsAnError(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeChunk(t, dir, "scalar", `return 42`)

	got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "scalar")
mus.response(tostring(ok) .. "|" .. tostring(err))`)
	if !strings.HasPrefix(got, "false|") || !strings.Contains(got, "must return a table") {
		t.Errorf("response = %q, want non-table error", got)
	}
}

func TestLoadSandboxed_ChunkErrorIsCatchable(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeChunk(t, dir, "broken", `error("broken chunk")`)

	got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "broken")
mus.response(tostring(ok) .. "|" .. tostring(err))`)
	if !strings.HasPrefix(got, "false|") || !strings.Contains(got, "broken chunk") {
		t.Errorf("response = %q, want the chunk's error", got)
	}
}

func TestLoadSandboxed_MissingChunkIsCatchable(t *testing.T) {
	engine, dir := sandboxedEngine(t)

	got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "nope")
mus.response(tostring(ok))`)
	if got != "false" {
		t.Errorf("ok = %q, want false", got)
	}
}

func TestLoadSandboxed_UnavailableStandardLibraryIsALuaError(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeChunk(t, dir, "plain", `return {}`)
	for _, library := range []string{"math", "string", "table"} {
		t.Run(library, func(t *testing.T) {
			got := probe(t, engine, dir, fmt.Sprintf(`%s = nil
local ok, err = pcall(mus.loadSandboxed, "plain")
mus.response(tostring(ok) .. "|" .. tostring(err))`, library))
			cause := fmt.Sprintf("sandbox standard library %q is unavailable", library)
			if !strings.HasPrefix(got, "false|") || !strings.Contains(got, cause) {
				t.Errorf("response = %q, want cause %q", got, cause)
			}
		})
	}
}

func TestLoadSandboxed_RejectsTheScriptsRootItself(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "toplevel", `return {}`)
	for _, chunkDir := range []string{".", "chunks/.."} {
		t.Run(chunkDir, func(t *testing.T) {
			engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil).
				WithSandboxedChunkDir(chunkDir)
			got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "toplevel")
mus.response(tostring(ok) .. "|" .. tostring(err))`)
			if !strings.HasPrefix(got, "false|") || !strings.Contains(got, "outside the scripts directory") {
				t.Errorf("response = %q, want containment error", got)
			}
		})
	}
}

func TestLoadSandboxed_RejectsAChunkSymlinkedOutside(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	outside := t.TempDir()
	writeScript(t, outside, "escape", `return {}`)
	if err := os.Symlink(filepath.Join(outside, "escape.lua"), filepath.Join(dir, "chunks", "escape.lua")); err != nil {
		t.Fatal(err)
	}

	got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "escape")
mus.response(tostring(ok) .. "|" .. tostring(err))`)
	if !strings.HasPrefix(got, "false|") || !strings.Contains(got, "outside the sandboxed directory") {
		t.Errorf("response = %q, want containment error", got)
	}
}

func TestLoadSandboxed_RejectsADirectorySymlinkedOutside(t *testing.T) {
	dir := setupScriptsDir(t)
	outside := t.TempDir()
	writeScript(t, outside, "escape", `return {}`)
	if err := os.Symlink(outside, filepath.Join(dir, "linked")); err != nil {
		t.Fatal(err)
	}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil).
		WithSandboxedChunkDir("linked")

	got := probe(t, engine, dir, `local ok, err = pcall(mus.loadSandboxed, "escape")
mus.response(tostring(ok) .. "|" .. tostring(err))`)
	if !strings.HasPrefix(got, "false|") || !strings.Contains(got, "outside the scripts directory") {
		t.Errorf("response = %q, want containment error", got)
	}
}

func TestLoadSandboxed_FollowsASymlinkThatStaysInside(t *testing.T) {
	engine, dir := sandboxedEngine(t)
	writeChunk(t, dir, "real", `return { name = "real" }`)
	if err := os.Symlink("real.lua", filepath.Join(dir, "chunks", "alias.lua")); err != nil {
		t.Fatal(err)
	}

	if got := probe(t, engine, dir, `mus.response(mus.loadSandboxed("alias").name)`); got != "real" {
		t.Errorf("response = %q, want real", got)
	}
}
