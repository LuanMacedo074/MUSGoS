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

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)
	if !engine.HasScript("echo") {
		t.Error("expected HasScript to return true")
	}
}

func TestHasScript_NotExists(t *testing.T) {
	dir := setupScriptsDir(t)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)
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

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

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

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

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

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

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

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

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

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

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
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, mockQueue, nil, nil, nil, nil, nil)

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

func TestExecute_SendMessage_EmptyRecipient(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "empty_recip", `mus.sendMessage("", "subj", "x")`)

	mockSender := &testutil.MockMessageSender{}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, mockSender, nil, nil, nil, nil)

	msg := &ports.ScriptMessage{
		Subject:  "empty_recip",
		SenderID: "user1",
		Content:  lingo.NewLVoid(),
	}

	_, err := engine.Execute(msg)
	if err == nil {
		t.Error("expected error when recipientID is empty")
	}
}

func TestExecute_SenderAccess(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "who", `mus.response(mus.getSender())`)

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, nil, nil)

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

func TestExecute_DBSetGetPlayerAttribute(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "dbtest", `
		mus.db.setPlayerAttribute("app1", "user1", "score", 42)
		local val = mus.db.getPlayerAttribute("app1", "user1", "score")
		mus.response(val)
	`)

	var stored lingo.LValue
	db := &testutil.MockDBAdapter{
		SetPlayerAttributeFunc: func(app, user, attr string, value lingo.LValue) error {
			stored = value
			return nil
		},
		GetPlayerAttributeFunc: func(app, user, attr string) (lingo.LValue, error) {
			return stored, nil
		},
	}

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, db, nil, nil, nil)

	msg := &ports.ScriptMessage{
		Subject:  "dbtest",
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

func TestExecute_ServerGetUserCount(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "srvtest", `
		local count = mus.server.getUserCount()
		mus.response(count)
	`)

	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("user1", "10.0.0.1")
	sessionStore.RegisterConnection("user2", "10.0.0.2")

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, sessionStore, nil)

	msg := &ports.ScriptMessage{
		Subject:  "srvtest",
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
	if intResult.Value != 2 {
		t.Errorf("expected 2, got %d", intResult.Value)
	}
}

func TestExecute_DBQueryBuilder(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "qbtest", `
		mus.db.table("items"):insert({ name = "sword", power = 10 })
		local row = mus.db.table("items"):where("name", "sword"):first()
		mus.response(row.power)
	`)

	// Create a real SQLite DB for the query builder test
	dbPath := filepath.Join(t.TempDir(), "qb_test.db")
	sqliteDB, err := outbound.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { sqliteDB.Close() })

	// Create a test table
	sqliteDB.CreateTable(ports.Table{
		Name: "items",
		Columns: []ports.Column{
			{Name: "id", Type: ports.ColInteger, IsPK: true, IsAutoIncr: true},
			{Name: "name", Type: ports.ColText, IsNotNull: true},
			{Name: "power", Type: ports.ColInteger},
		},
	})

	qb := sqliteDB.QueryBuilder()
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, qb, nil, nil)

	msg := &ports.ScriptMessage{
		Subject:  "qbtest",
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
	if intResult.Value != 10 {
		t.Errorf("expected 10, got %d", intResult.Value)
	}
}
