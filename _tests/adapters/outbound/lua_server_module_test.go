package outbound_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

func TestServer_IsOnline(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "online", `
if mus.server.isOnline(mus.getContent()) then mus.response(1) else mus.response(0) end
`)

	ss := testutil.NewMockSessionStore()
	ss.RegisterConnection("alice", "10.0.0.1")

	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, ss, nil)

	check := func(name string, want int32) {
		t.Helper()
		res, err := engine.Execute(&ports.ScriptMessage{Subject: "online", SenderID: "alice", Content: lingo.NewLString(name)})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		n, ok := res.Content.(*lingo.LInteger)
		if !ok || n.Value != want {
			t.Fatalf("isOnline(%q): expected %d, got %v", name, want, res.Content)
		}
	}
	check("alice", 1)
	check("bob", 0)
}

func TestServer_Broadcast_ReachesAllConnections(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "bcast", `mus.server.broadcast("News", "hello all")`)

	ss := testutil.NewMockSessionStore()
	ss.RegisterConnection("alice", "10.0.0.1")
	ss.RegisterConnection("bob", "10.0.0.2")
	ss.RegisterConnection("carol", "10.0.0.3")

	sender := &testutil.MockMessageSender{}
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, sender, nil, nil, ss, nil)

	// Invoked by an arbitrary sender, but broadcasts must go out AS system.script
	// (the client only renders "Broadcast" from system.script; the human name
	// lives in the content, set by the calling Lua script).
	_, err := engine.Execute(&ports.ScriptMessage{Subject: "bcast", SenderID: "some-player", Content: lingo.NewLVoid()})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(sender.Calls) != 3 {
		t.Fatalf("expected 3 broadcast deliveries, got %d", len(sender.Calls))
	}
	recipients := map[string]bool{}
	for _, c := range sender.Calls {
		recipients[c.RecipientID] = true
		if c.Subject != "News" {
			t.Errorf("expected subject News, got %q", c.Subject)
		}
		if c.SenderID != "system.script" {
			t.Errorf("expected broadcast senderID system.script, got %q", c.SenderID)
		}
		if s, ok := c.Content.(*lingo.LString); !ok || s.Value != "hello all" {
			t.Errorf("expected content 'hello all', got %v", c.Content)
		}
	}
	for _, want := range []string{"alice", "bob", "carol"} {
		if !recipients[want] {
			t.Errorf("expected a broadcast delivery to %q", want)
		}
	}
}

func TestServer_Broadcast_NoOpWithoutSender(t *testing.T) {
	dir := setupScriptsDir(t)
	writeScript(t, dir, "bcast", `mus.server.broadcast("News", "x")`)

	ss := testutil.NewMockSessionStore()
	ss.RegisterConnection("alice", "10.0.0.1")

	// sender is nil — broadcast must be a safe no-op (no panic).
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 5, nil, nil, nil, nil, ss, nil)
	if _, err := engine.Execute(&ports.ScriptMessage{Subject: "bcast", SenderID: "sys", Content: lingo.NewLVoid()}); err != nil {
		t.Fatalf("execute: %v", err)
	}
}
