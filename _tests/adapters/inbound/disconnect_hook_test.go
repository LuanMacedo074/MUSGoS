package inbound_test

import (
	"testing"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// recordingEngine captures the last Execute call for assertions.
type recordingEngine struct {
	has      bool
	executed *ports.ScriptMessage
	calls    int
}

func (e *recordingEngine) HasScript(subject string) bool { return e.has }
func (e *recordingEngine) Execute(msg *ports.ScriptMessage) (*ports.ScriptResult, error) {
	e.calls++
	e.executed = msg
	return &ports.ScriptResult{Content: lingo.NewLVoid()}, nil
}

func TestDisconnectHook_FiresWithUserID(t *testing.T) {
	eng := &recordingEngine{has: true}
	hook := inbound.NewDisconnectFlushHook(eng, "users/onDisconnect", nil)
	if hook == nil {
		t.Fatal("expected a non-nil hook")
	}

	hook("hero")

	if eng.calls != 1 {
		t.Fatalf("expected 1 execute, got %d", eng.calls)
	}
	if eng.executed.Subject != "users/onDisconnect" {
		t.Errorf("subject = %q, want users/onDisconnect", eng.executed.Subject)
	}
	if eng.executed.SenderID != "hero" {
		t.Errorf("senderID = %q, want hero", eng.executed.SenderID)
	}
}

func TestDisconnectHook_NoOpWhenScriptAbsent(t *testing.T) {
	eng := &recordingEngine{has: false}
	hook := inbound.NewDisconnectFlushHook(eng, "users/onDisconnect", nil)
	hook("hero")
	if eng.calls != 0 {
		t.Fatalf("expected no execute when script absent, got %d", eng.calls)
	}
}

func TestDisconnectHook_NoOpForEmptyID(t *testing.T) {
	eng := &recordingEngine{has: true}
	hook := inbound.NewDisconnectFlushHook(eng, "users/onDisconnect", nil)
	hook("")
	if eng.calls != 0 {
		t.Fatalf("expected no execute for empty id, got %d", eng.calls)
	}
}

func TestDisconnectHook_DisabledReturnsNil(t *testing.T) {
	if inbound.NewDisconnectFlushHook(nil, "users/onDisconnect", nil) != nil {
		t.Error("expected nil hook when engine is nil")
	}
	eng := &recordingEngine{has: true}
	if inbound.NewDisconnectFlushHook(eng, "", nil) != nil {
		t.Error("expected nil hook when subject is empty")
	}
}
