package smus_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/smus"
)

func TestExtractMUSMsgHeaderString_Even(t *testing.T) {
	str := "hi" // even length
	buf := make([]byte, 4+len(str))
	binary.BigEndian.PutUint32(buf, uint32(len(str)))
	copy(buf[4:], str)

	m := &smus.MUSMsgHeaderString{}
	consumed, err := m.ExtractMUSMsgHeaderString(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Value != str {
		t.Errorf("Value = %q, want %q", m.Value, str)
	}
	if m.Length != len(str) {
		t.Errorf("Length = %d, want %d", m.Length, len(str))
	}
	// even: 4 + 2 = 6
	if consumed != 6 {
		t.Errorf("consumed = %d, want 6", consumed)
	}
}

func TestExtractMUSMsgHeaderString_Odd(t *testing.T) {
	str := "hey" // odd length
	buf := make([]byte, 4+len(str)+1) // +1 for padding
	binary.BigEndian.PutUint32(buf, uint32(len(str)))
	copy(buf[4:], str)

	m := &smus.MUSMsgHeaderString{}
	consumed, err := m.ExtractMUSMsgHeaderString(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Value != str {
		t.Errorf("Value = %q, want %q", m.Value, str)
	}
	// odd: 4 + 3 + 1 = 8
	if consumed != 8 {
		t.Errorf("consumed = %d, want 8", consumed)
	}
}

func TestExtractMUSMsgHeaderString_TooShort(t *testing.T) {
	buf := []byte{0x00, 0x00} // only 2 bytes, need 4 for length
	m := &smus.MUSMsgHeaderString{}
	_, err := m.ExtractMUSMsgHeaderString(buf, 0)
	if err == nil {
		t.Error("expected error for too-short data")
	}
}

func TestExtractMUSMsgHeaderString_Empty(t *testing.T) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 0) // length = 0

	m := &smus.MUSMsgHeaderString{}
	consumed, err := m.ExtractMUSMsgHeaderString(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Value != "" {
		t.Errorf("Value = %q, want empty", m.Value)
	}
	if m.Length != 0 {
		t.Errorf("Length = %d, want 0", m.Length)
	}
	if consumed != 4 {
		t.Errorf("consumed = %d, want 4", consumed)
	}
}
