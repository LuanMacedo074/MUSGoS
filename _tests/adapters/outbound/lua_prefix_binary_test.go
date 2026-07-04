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

// Prefix subjects: a dash-suffixed subject with no exact script falls back to
// the longest on-disk prefix, and the script reads the full original subject
// via mus.getSubject().
func TestExecute_PrefixSubjectRouting(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "painting"), 0755); err != nil {
		t.Fatal(err)
	}
	script := `mus.response(mus.getSubject())`
	if err := os.WriteFile(filepath.Join(dir, "painting", "saveLayer1.lua"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)

	if !engine.HasScript("painting/saveLayer1-42") {
		t.Fatal("expected prefix subject to resolve")
	}
	if engine.HasScript("painting/other-42") {
		t.Fatal("unrelated prefix subject must not resolve")
	}

	res, err := engine.Execute(&ports.ScriptMessage{
		Subject:  "painting/saveLayer1-42",
		SenderID: "tester",
		Content:  lingo.NewLString("x"),
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	got, ok := res.Content.(*lingo.LString)
	if !ok || got.Value != "painting/saveLayer1-42" {
		t.Fatalf("getSubject: expected full original subject, got %#v", res.Content)
	}
}

// Exact scripts always win over prefix fallback.
func TestExecute_PrefixSubjectExactWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.lua"), []byte(`mus.response("prefix")`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a-1.lua"), []byte(`mus.response("exact")`), 0644); err != nil {
		t.Fatal(err)
	}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)
	res, err := engine.Execute(&ports.ScriptMessage{Subject: "a-1", SenderID: "t", Content: lingo.NewLString("x")})
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.Content.(*lingo.LString); !ok || s.Value != "exact" {
		t.Fatalf("expected exact script to win, got %#v", res.Content)
	}
}

// Media/picture values must round-trip through Lua without losing the lingo
// type: they arrive as a tagged table and mus.response(table) rebuilds them.
func TestExecute_BinaryContentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	script := `
local m = mus.getContent()
assert(type(m) == "table", "media should be a table")
assert(m.__lingo == "media", "media tag")
assert(m.data == "\1\2\255\0\254", "bytes intact")
mus.response(m)
`
	if err := os.WriteFile(filepath.Join(dir, "echo.lua"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)

	payload := []byte{1, 2, 255, 0, 254}
	res, err := engine.Execute(&ports.ScriptMessage{
		Subject:  "echo",
		SenderID: "t",
		Content:  lingo.NewLMedia(payload),
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	media, ok := res.Content.(*lingo.LMedia)
	if !ok {
		t.Fatalf("expected LMedia back, got %#v", res.Content)
	}
	if string(media.Data) != string(payload) {
		t.Fatalf("media bytes changed: %v", media.Data)
	}
}

func TestExecute_PictureContentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	script := `mus.response(mus.getContent())`
	if err := os.WriteFile(filepath.Join(dir, "echo.lua"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)

	payload := []byte{0, 9, 8, 7}
	res, err := engine.Execute(&ports.ScriptMessage{
		Subject:  "echo",
		SenderID: "t",
		Content:  lingo.NewLPicture(payload),
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	pic, ok := res.Content.(*lingo.LPicture)
	if !ok {
		t.Fatalf("expected LPicture back, got %#v", res.Content)
	}
	if string(pic.Data) != string(payload) {
		t.Fatalf("picture bytes changed: %v", pic.Data)
	}
}
