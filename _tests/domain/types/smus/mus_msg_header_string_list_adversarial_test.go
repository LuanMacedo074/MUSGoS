package smus_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/smus"
)

// A wire count far larger than the remaining bytes must be rejected with an
// error instead of triggering a multi-GB pre-allocation (pre-auth DoS, B1).
func TestExtractStringList_HugeCount_Rejected(t *testing.T) {
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data[0:], 0x7FFFFFFF) // count claims ~2.1 billion strings
	// remaining after the 4-byte count is 4 bytes → at most 1 element could fit.

	var list smus.MUSMsgHeaderStringList
	consumed, err := list.ExtractMUSMsgHeaderStringList(data, 0)
	if err == nil {
		t.Fatal("expected error for oversized list count, got nil")
	}
	if consumed != 0 {
		t.Errorf("expected 0 bytes consumed on error, got %d", consumed)
	}
	if len(list.Strings) != 0 {
		t.Errorf("expected no strings allocated, got %d", len(list.Strings))
	}
}

// A well-formed list must still parse after the bound check.
func TestExtractStringList_ValidRoundtrip(t *testing.T) {
	var buf []byte
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, 2)
	buf = append(buf, count...)
	for _, s := range []string{"ab", "cd"} {
		l := make([]byte, 4)
		binary.BigEndian.PutUint32(l, uint32(len(s)))
		buf = append(buf, l...)
		buf = append(buf, []byte(s)...)
	}

	var list smus.MUSMsgHeaderStringList
	if _, err := list.ExtractMUSMsgHeaderStringList(buf, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Strings) != 2 || list.Strings[0].Value != "ab" || list.Strings[1].Value != "cd" {
		t.Errorf("unexpected parse result: %+v", list.Strings)
	}
}
