package smus_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/smus"

	"fsos-server/_tests/testutil"
)

// buildValidMUSMessage constructs a minimal valid MUS message.
// Structure: header(2) + contentSize(4) + errCode(4) + timestamp(4) + subject + sender + recipients
func buildValidMUSMessage(subject, sender string, recipients []string) []byte {
	var payload []byte

	// errCode
	errCode := make([]byte, 4)
	payload = append(payload, errCode...)

	// timestamp
	ts := make([]byte, 4)
	payload = append(payload, ts...)

	// subject
	payload = append(payload, headerString(subject)...)

	// sender
	payload = append(payload, headerString(sender)...)

	// recipients list
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, uint32(len(recipients)))
	payload = append(payload, countBytes...)
	for _, r := range recipients {
		payload = append(payload, headerString(r)...)
	}

	// Now build the full message
	var msg []byte
	// MUS header
	msg = append(msg, 0x72, 0x00)
	// content size
	contentSize := make([]byte, 4)
	binary.BigEndian.PutUint32(contentSize, uint32(len(payload)))
	msg = append(msg, contentSize...)
	// payload
	msg = append(msg, payload...)

	return msg
}

func headerString(s string) []byte {
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

func TestParseMUSMessage_Valid(t *testing.T) {
	raw := buildValidMUSMessage("Test", "user1", []string{"user2"})

	msg, err := smus.ParseMUSMessage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Subject.Value != "Test" {
		t.Errorf("Subject = %q, want %q", msg.Subject.Value, "Test")
	}
	if msg.SenderID.Value != "user1" {
		t.Errorf("SenderID = %q, want %q", msg.SenderID.Value, "user1")
	}
	if msg.RecptID.Count != 1 {
		t.Errorf("RecptID.Count = %d, want 1", msg.RecptID.Count)
	}
	if msg.RecptID.Strings[0].Value != "user2" {
		t.Errorf("RecptID.Strings[0] = %q, want %q", msg.RecptID.Strings[0].Value, "user2")
	}
}

func TestParseMUSMessage_TooShort(t *testing.T) {
	raw := []byte{0x72, 0x00, 0x00} // only 3 bytes
	_, err := smus.ParseMUSMessage(raw)
	if err == nil {
		t.Error("expected error for short message")
	}
}

func TestParseMUSMessage_InvalidHeader(t *testing.T) {
	raw := make([]byte, 20)
	raw[0] = 0xFF // wrong header
	raw[1] = 0xFF

	_, err := smus.ParseMUSMessage(raw)
	if err == nil {
		t.Error("expected error for invalid header")
	}
}

func TestParseMUSMessage_Truncated(t *testing.T) {
	var msg []byte
	msg = append(msg, 0x72, 0x00)
	// claim content size of 1000 but provide very little data
	contentSize := make([]byte, 4)
	binary.BigEndian.PutUint32(contentSize, 1000)
	msg = append(msg, contentSize...)
	msg = append(msg, 0x00, 0x00, 0x00, 0x00) // just 4 bytes of payload

	_, err := smus.ParseMUSMessage(msg)
	if err == nil {
		t.Error("expected error for truncated message")
	}
}

func TestParseMUSMessageWithDecryption_Logon(t *testing.T) {
	raw := buildValidMUSMessage("Logon", "user1", []string{"system"})

	// Append some content bytes after the headers
	// We need to recalculate content size to include content
	content := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x2A} // VtInteger + value 42

	// Rebuild with content
	raw = buildValidMUSMessageWithContent("Logon", "user1", []string{"system"}, content)

	cipher := &testutil.MockCipher{}
	msg, err := smus.ParseMUSMessageWithDecryption(raw, cipher)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cipher.DecryptCalls != 1 {
		t.Errorf("DecryptCalls = %d, want 1", cipher.DecryptCalls)
	}
	_ = msg
}

func TestParseMUSMessageWithDecryption_NonLogon(t *testing.T) {
	content := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x2A}
	raw := buildValidMUSMessageWithContent("Chat", "user1", []string{"user2"}, content)

	cipher := &testutil.MockCipher{}
	_, err := smus.ParseMUSMessageWithDecryption(raw, cipher)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cipher.DecryptCalls != 0 {
		t.Errorf("DecryptCalls = %d, want 0", cipher.DecryptCalls)
	}
}

func TestParseMUSMessage_NoContent(t *testing.T) {
	raw := buildValidMUSMessage("Ping", "user1", []string{"user2"})

	msg, err := smus.ParseMUSMessage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// MsgContent should be nil when there's no remaining data after headers
	// (or could be a void depending on implementation — we just check no panic)
	_ = msg
}

func TestMUSMessage_GetBytes_RoundTrip(t *testing.T) {
	// Build a message with content, serialize it, parse it back
	content := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x2A} // VtInteger(42)
	raw := buildValidMUSMessageWithContent("Chat", "user1", []string{"user2", "user3"}, content)

	msg, err := smus.ParseMUSMessage(raw)
	if err != nil {
		t.Fatalf("initial parse: %v", err)
	}

	// Serialize and re-parse
	serialized := msg.GetBytes()
	reparsed, err := smus.ParseMUSMessage(serialized)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	if reparsed.Subject.Value != "Chat" {
		t.Errorf("Subject = %q, want %q", reparsed.Subject.Value, "Chat")
	}
	if reparsed.SenderID.Value != "user1" {
		t.Errorf("SenderID = %q, want %q", reparsed.SenderID.Value, "user1")
	}
	if reparsed.RecptID.Count != 2 {
		t.Errorf("RecptID.Count = %d, want 2", reparsed.RecptID.Count)
	}
	if reparsed.MsgContent.ToInteger() != 42 {
		t.Errorf("MsgContent = %d, want 42", reparsed.MsgContent.ToInteger())
	}
}

func buildValidMUSMessageWithContent(subject, sender string, recipients []string, content []byte) []byte {
	var payload []byte

	// errCode
	errCode := make([]byte, 4)
	payload = append(payload, errCode...)

	// timestamp
	ts := make([]byte, 4)
	payload = append(payload, ts...)

	// subject
	payload = append(payload, headerString(subject)...)

	// sender
	payload = append(payload, headerString(sender)...)

	// recipients list
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, uint32(len(recipients)))
	payload = append(payload, countBytes...)
	for _, r := range recipients {
		payload = append(payload, headerString(r)...)
	}

	// content
	payload = append(payload, content...)

	// Build full message
	var msg []byte
	msg = append(msg, 0x72, 0x00)
	contentSize := make([]byte, 4)
	binary.BigEndian.PutUint32(contentSize, uint32(len(payload)))
	msg = append(msg, contentSize...)
	msg = append(msg, payload...)

	return msg
}
