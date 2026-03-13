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

func setupScriptsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func writeScript(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name+".lua"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
}

func TestHasScript_Exists(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "echo", "return 1")

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)
	if !engine.HasScript("echo") {
		t.Error("expected HasScript to return true")
	}
}

func TestHasScript_NotExists(t *testing.T) {
	dir := setupScriptsDir(t)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)
	if engine.HasScript("nonexistent") {
		t.Error("expected HasScript to return false")
	}
}

func TestExecute_GetSenderAndContent(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "test", `
local sender = mus.getSender()
local content = mus.getContent()
mus.response(content)
`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)

	msg := &ports.ScriptMessage{
		Subject:  "test",
		SenderID: "user1",
		Content:  lingo.NewLString("hello"),
	}

	result, err := engine.Execute(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	strResult, ok := result.Content.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", result.Content)
	}
	if strResult.Value != "hello" {
		t.Errorf("expected \"hello\", got %q", strResult.Value)
	}
}

func TestExecute_ResponseWithValue(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "ret", `mus.response(42)`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)

	msg := &ports.ScriptMessage{
		Subject:  "ret",
		SenderID: "user1",
		Content:  lingo.NewLVoid(),
	}

	result, err := engine.Execute(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intResult, ok := result.Content.(*lingo.LInteger)
	if !ok {
		t.Fatalf("expected *LInteger, got %T", result.Content)
	}
	if intResult.Value != 42 {
		t.Errorf("expected 42, got %d", intResult.Value)
	}
}

func TestExecute_NoResponse_ReturnsVoid(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "noop", `local x = 1 + 1`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)

	msg := &ports.ScriptMessage{
		Subject:  "noop",
		SenderID: "user1",
		Content:  lingo.NewLVoid(),
	}

	result, err := engine.Execute(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result.Content.(*lingo.LVoid); !ok {
		t.Errorf("expected *LVoid when no mus.response() called, got %T", result.Content)
	}
}

func TestExecute_LuaError(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "bad", `error("something went wrong")`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)

	msg := &ports.ScriptMessage{
		Subject:  "bad",
		SenderID: "user1",
		Content:  lingo.NewLVoid(),
	}

	_, err := engine.Execute(msg)
	if err == nil {
		t.Error("expected error for broken script")
	}
}

func TestExecute_ScriptNotFound(t *testing.T) {
	dir := setupScriptsDir(t)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)

	msg := &ports.ScriptMessage{
		Subject:  "missing",
		SenderID: "user1",
		Content:  lingo.NewLVoid(),
	}

	_, err := engine.Execute(msg)
	if err == nil {
		t.Error("expected error for missing script")
	}
}

func TestExecute_Publish(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "pub", `mus.publish("my.topic", "hello world")`)

	mockQueue := testutil.NewMockMessageQueue()
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, mockQueue)

	msg := &ports.ScriptMessage{
		Subject:  "pub",
		SenderID: "user1",
		Content:  lingo.NewLVoid(),
	}

	_, err := engine.Execute(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockQueue.PublishCalls) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(mockQueue.PublishCalls))
	}
	call := mockQueue.PublishCalls[0]
	if call.Topic != "my.topic" {
		t.Errorf("topic = %q, want %q", call.Topic, "my.topic")
	}
	// Payload is Lingo-encoded (LuaToLValue -> GetBytes), so decode it back
	parsed := lingo.FromRawBytes(call.Payload, 0)
	strVal, ok := parsed.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString payload, got %T", parsed)
	}
	if strVal.Value != "hello world" {
		t.Errorf("payload = %q, want %q", strVal.Value, "hello world")
	}
}

func TestExecute_SenderAccess(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "who", `mus.response(mus.getSender())`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil)

	msg := &ports.ScriptMessage{
		Subject:  "who",
		SenderID: "player42",
		Content:  lingo.NewLVoid(),
	}

	result, err := engine.Execute(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	strResult, ok := result.Content.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", result.Content)
	}
	if strResult.Value != "player42" {
		t.Errorf("expected \"player42\", got %q", strResult.Value)
	}
}
