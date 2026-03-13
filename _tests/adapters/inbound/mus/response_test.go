package mus_test

import (
	"testing"

	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func TestNewResponse_BuildsCorrectMessage(t *testing.T) {
	content := lingo.NewLInteger(42)
	resp := mus.NewResponse("Logon", "System", []string{"testuser"}, smus.ErrNoError, content)

	if resp.Subject.Value != "Logon" {
		t.Errorf("Subject = %q, want %q", resp.Subject.Value, "Logon")
	}
	if resp.SenderID.Value != "System" {
		t.Errorf("SenderID = %q, want %q", resp.SenderID.Value, "System")
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if resp.RecptID.Count != 1 {
		t.Errorf("RecptID.Count = %d, want 1", resp.RecptID.Count)
	}
	if resp.RecptID.Strings[0].Value != "testuser" {
		t.Errorf("RecptID[0] = %q, want %q", resp.RecptID.Strings[0].Value, "testuser")
	}
	if resp.MsgContent.ToInteger() != 42 {
		t.Errorf("MsgContent = %d, want 42", resp.MsgContent.ToInteger())
	}
}

func TestNewResponse_MultipleRecipients(t *testing.T) {
	resp := mus.NewResponse("Chat", "user1", []string{"user2", "user3"}, smus.ErrNoError, lingo.NewLVoid())

	if resp.RecptID.Count != 2 {
		t.Errorf("RecptID.Count = %d, want 2", resp.RecptID.Count)
	}
	if resp.RecptID.Strings[0].Value != "user2" {
		t.Errorf("RecptID[0] = %q, want %q", resp.RecptID.Strings[0].Value, "user2")
	}
	if resp.RecptID.Strings[1].Value != "user3" {
		t.Errorf("RecptID[1] = %q, want %q", resp.RecptID.Strings[1].Value, "user3")
	}
}

func TestNewResponse_RoundTrip(t *testing.T) {
	content := lingo.NewLString("hello")
	resp := mus.NewResponse("Test", "sender", []string{"recpt"}, smus.ErrNoError, content)

	rawBytes := resp.GetBytes()

	parsed, err := smus.ParseMUSMessage(rawBytes)
	if err != nil {
		t.Fatalf("failed to parse response bytes: %v", err)
	}
	if parsed.Subject.Value != "Test" {
		t.Errorf("parsed Subject = %q, want %q", parsed.Subject.Value, "Test")
	}
	if parsed.SenderID.Value != "sender" {
		t.Errorf("parsed SenderID = %q, want %q", parsed.SenderID.Value, "sender")
	}
	if parsed.RecptID.Count != 1 || parsed.RecptID.Strings[0].Value != "recpt" {
		t.Errorf("parsed RecptID unexpected")
	}
}
