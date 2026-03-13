package inbound_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

func buildValidSMUSMessage(subject, sender string, recipients []string) []byte {
	var payload []byte

	// errCode
	payload = append(payload, make([]byte, 4)...)
	// timestamp
	payload = append(payload, make([]byte, 4)...)

	// subject
	payload = append(payload, hdrString(subject)...)
	// sender
	payload = append(payload, hdrString(sender)...)

	// recipients
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, uint32(len(recipients)))
	payload = append(payload, countBytes...)
	for _, r := range recipients {
		payload = append(payload, hdrString(r)...)
	}

	var msg []byte
	msg = append(msg, 0x72, 0x00)
	contentSize := make([]byte, 4)
	binary.BigEndian.PutUint32(contentSize, uint32(len(payload)))
	msg = append(msg, contentSize...)
	msg = append(msg, payload...)
	return msg
}

func hdrString(s string) []byte {
	var buf []byte
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(s)))
	buf = append(buf, lenBytes...)
	buf = append(buf, []byte(s)...)
	if len(s)%2 != 0 {
		buf = append(buf, 0x00)
	}
	return buf
}

func TestSMUSHandler_HandleRawMessage_Valid(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}

	handler := inbound.NewSMUSHandler(logger, cipher, nil, nil, nil, nil, nil)
	raw := buildValidSMUSMessage("Test", "user1", []string{"user2"})

	_, err := handler.HandleRawMessage("client-1", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Logger should have been called
	if len(logger.Messages) == 0 {
		t.Error("expected logger to receive calls")
	}
}

func TestSMUSHandler_HandleRawMessage_Invalid(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}

	handler := inbound.NewSMUSHandler(logger, cipher, nil, nil, nil, nil, nil)

	_, err := handler.HandleRawMessage("client-1", []byte{0xFF, 0xFF})
	if err == nil {
		t.Error("expected error for invalid message")
	}
}

func TestSMUSHandler_SystemScript_ExecutesScript(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}

	var executedSubject string
	scriptEngine := &testutil.MockScriptEngine{
		HasScriptFunc: func(subject string) bool {
			return subject == "QueryCreate"
		},
		ExecuteFunc: func(msg *ports.ScriptMessage) (*ports.ScriptResult, error) {
			executedSubject = msg.Subject
			return &ports.ScriptResult{Content: lingo.NewLString("ok")}, nil
		},
	}

	handler := inbound.NewSMUSHandler(logger, cipher, scriptEngine, nil, nil, nil, nil)
	raw := buildValidSMUSMessage("QueryCreate", "user1", []string{"system.script"})

	resp, err := handler.HandleRawMessage("client-1", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executedSubject != "QueryCreate" {
		t.Errorf("script subject = %q, want %q", executedSubject, "QueryCreate")
	}
	if resp == nil {
		t.Fatal("expected response bytes when script returns content")
	}
}

func TestSMUSHandler_SystemScript_NoScript(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}

	scriptEngine := &testutil.MockScriptEngine{
		HasScriptFunc: func(subject string) bool {
			return false
		},
	}

	handler := inbound.NewSMUSHandler(logger, cipher, scriptEngine, nil, nil, nil, nil)
	raw := buildValidSMUSMessage("NonExistent", "user1", []string{"system.script"})

	resp, err := handler.HandleRawMessage("client-1", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response for missing script")
	}
}

func TestSMUSHandler_NonSystemScript_DoesNotExecute(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}

	executed := false
	scriptEngine := &testutil.MockScriptEngine{
		HasScriptFunc: func(subject string) bool {
			return true
		},
		ExecuteFunc: func(msg *ports.ScriptMessage) (*ports.ScriptResult, error) {
			executed = true
			return &ports.ScriptResult{Content: lingo.NewLVoid()}, nil
		},
	}

	handler := inbound.NewSMUSHandler(logger, cipher, scriptEngine, nil, nil, nil, nil)
	raw := buildValidSMUSMessage("QueryCreate", "user1", []string{"someuser"})

	_, err := handler.HandleRawMessage("client-1", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executed {
		t.Error("script should not execute when recipient is not system.script")
	}
}
