package outbound_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

func encodeViaScript(t *testing.T, body string) string {
	t.Helper()
	dir := setupScriptsDir(t)
	writeScript(t, dir, "jsontest", body)
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil, nil)
	result, err := engine.Execute(&ports.ScriptMessage{Subject: "jsontest", SenderID: "u1", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	strResult, ok := result.Content.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", result.Content)
	}
	return strResult.Value
}

func TestExecute_JsonArrayEmptyEncodesAsArray(t *testing.T) {
	if got := encodeViaScript(t, `mus.response(mus.json.encode(mus.json.array()))`); got != "[]" {
		t.Errorf("expected [], got %q", got)
	}
}

func TestExecute_JsonArrayNestedEmptyEncodesAsArray(t *testing.T) {
	if got := encodeViaScript(t, `mus.response(mus.json.encode({items = mus.json.array()}))`); got != `{"items":[]}` {
		t.Errorf(`expected {"items":[]}, got %q`, got)
	}
}

func TestExecute_JsonArrayFilledEncodesAsArray(t *testing.T) {
	got := encodeViaScript(t, `local a = mus.json.array()
table.insert(a, 1)
table.insert(a, "x")
mus.response(mus.json.encode(a))`)
	if got != `[1,"x"]` {
		t.Errorf(`expected [1,"x"], got %q`, got)
	}
}
