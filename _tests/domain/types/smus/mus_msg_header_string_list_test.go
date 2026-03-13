package smus_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/smus"
)

func buildHeaderStringListBytes(strs []string) []byte {
	var buf []byte
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, uint32(len(strs)))
	buf = append(buf, countBytes...)

	for _, s := range strs {
		strLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(strLenBytes, uint32(len(s)))
		buf = append(buf, strLenBytes...)
		buf = append(buf, []byte(s)...)
		if len(s)%2 != 0 {
			buf = append(buf, 0x00) // padding
		}
	}
	return buf
}

func TestExtractMUSMsgHeaderStringList_Single(t *testing.T) {
	buf := buildHeaderStringListBytes([]string{"ab"})

	m := &smus.MUSMsgHeaderStringList{}
	consumed, err := m.ExtractMUSMsgHeaderStringList(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Count != 1 {
		t.Errorf("Count = %d, want 1", m.Count)
	}
	if m.Strings[0].Value != "ab" {
		t.Errorf("Strings[0].Value = %q, want %q", m.Strings[0].Value, "ab")
	}
	if consumed != 4+4+2 { // count + strlen + "ab"
		t.Errorf("consumed = %d, want %d", consumed, 10)
	}
}

func TestExtractMUSMsgHeaderStringList_Multiple(t *testing.T) {
	buf := buildHeaderStringListBytes([]string{"hi", "hey"})

	m := &smus.MUSMsgHeaderStringList{}
	_, err := m.ExtractMUSMsgHeaderStringList(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Count != 2 {
		t.Errorf("Count = %d, want 2", m.Count)
	}
	if m.Strings[0].Value != "hi" {
		t.Errorf("Strings[0].Value = %q, want %q", m.Strings[0].Value, "hi")
	}
	if m.Strings[1].Value != "hey" {
		t.Errorf("Strings[1].Value = %q, want %q", m.Strings[1].Value, "hey")
	}
}

func TestExtractMUSMsgHeaderStringList_Empty(t *testing.T) {
	buf := buildHeaderStringListBytes([]string{})

	m := &smus.MUSMsgHeaderStringList{}
	consumed, err := m.ExtractMUSMsgHeaderStringList(buf, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Count != 0 {
		t.Errorf("Count = %d, want 0", m.Count)
	}
	if consumed != 4 {
		t.Errorf("consumed = %d, want 4", consumed)
	}
}

func TestExtractMUSMsgHeaderStringList_Truncated(t *testing.T) {
	// count=2 but only provide data for 0 strings
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 2)

	m := &smus.MUSMsgHeaderStringList{}
	_, err := m.ExtractMUSMsgHeaderStringList(buf, 0)
	if err == nil {
		t.Error("expected error for truncated data")
	}
}
