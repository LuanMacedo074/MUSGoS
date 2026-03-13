package inbound_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
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

	handler := inbound.NewSMUSHandler(logger, cipher)
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

	handler := inbound.NewSMUSHandler(logger, cipher)

	_, err := handler.HandleRawMessage("client-1", []byte{0xFF, 0xFF})
	if err == nil {
		t.Error("expected error for invalid message")
	}
}
