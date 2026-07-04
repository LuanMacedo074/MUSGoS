package inbound_test

import (
	"net"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/domain/ports"
)

// B4: a non-Logon message whose wire SenderID claims a different user must be
// dispatched under the connection's own identity, never the spoofed SenderID
// (otherwise the caller inherits the victim's command level).
func TestSMUSHandler_SpoofedSenderID_DispatchesUnderConnection(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}
	connWriter := &testutil.MockConnectionWriter{}

	var capturedSender string
	scriptEngine := &testutil.MockScriptEngine{
		HasScriptFunc: func(subject string) bool { return subject == "move" },
		ExecuteFunc: func(msg *ports.ScriptMessage) (*ports.ScriptResult, error) {
			capturedSender = msg.SenderID
			return &ports.ScriptResult{Content: nil}, nil
		},
	}

	dispatcher := newSMUSTestDispatcher(scriptEngine, connWriter)
	handler := inbound.NewSMUSHandler(logger, cipher, dispatcher, false)

	// Delivered on connection "attacker" but claims SenderID "victim". Routed to
	// the script engine (recipient "system.script"), whose ScriptMessage.SenderID
	// must reflect the connection, not the spoofed wire SenderID.
	msg := buildValidSMUSMessage("move", "victim", []string{"system.script"})
	if _, err := handler.HandleRawMessage("attacker", msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedSender != "attacker" {
		t.Errorf("dispatch senderID = %q, want %q (wire SenderID must be ignored)", capturedSender, "attacker")
	}
}

// H3: RemapClientID must refuse to overwrite a userID already bound to another
// connection, so a second client cannot hijack the first's mapping.
func TestConnPool_RemapClientID_RefusesClobber(t *testing.T) {
	pool := inbound.NewConnPool()
	a, ca := net.Pipe()
	defer a.Close()
	defer ca.Close()
	b, cb := net.Pipe()
	defer b.Close()
	defer cb.Close()

	pool.Register(a, "connA")
	pool.Register(b, "connB")

	if !pool.RemapClientID("connA", "user1") {
		t.Fatal("first remap onto a free id should succeed")
	}
	if pool.RemapClientID("connB", "user1") {
		t.Error("remap onto an already-bound userID should return false")
	}
	if id := pool.CurrentID(a); id != "user1" {
		t.Errorf("conn A CurrentID = %q, want user1", id)
	}
	if id := pool.CurrentID(b); id != "connB" {
		t.Errorf("conn B CurrentID = %q, want connB (unchanged)", id)
	}
}
