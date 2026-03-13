package lingo_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLString_NewAndValue(t *testing.T) {
	v := lingo.NewLString("hello")
	if v.Value != "hello" {
		t.Errorf("Value = %q, want %q", v.Value, "hello")
	}
	if v.GetType() != lingo.VtString {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtString)
	}
}

func TestLString_ExtractFromBytes_EvenLength(t *testing.T) {
	// "hi" is 2 bytes (even) — no padding
	str := "hi"
	buf := make([]byte, 4+len(str))
	binary.BigEndian.PutUint32(buf[0:], uint32(len(str)))
	copy(buf[4:], str)

	v := &lingo.LString{}
	consumed := v.ExtractFromBytes(buf, 0)
	if v.Value != str {
		t.Errorf("Value = %q, want %q", v.Value, str)
	}
	// Even length: 4 (length prefix) + 2 (string) = 6
	if consumed != 6 {
		t.Errorf("consumed = %d, want 6", consumed)
	}
}

func TestLString_ExtractFromBytes_OddLength(t *testing.T) {
	// "hey" is 3 bytes (odd) — 1 byte padding
	str := "hey"
	buf := make([]byte, 4+len(str)+1) // +1 for padding
	binary.BigEndian.PutUint32(buf[0:], uint32(len(str)))
	copy(buf[4:], str)

	v := &lingo.LString{}
	consumed := v.ExtractFromBytes(buf, 0)
	if v.Value != str {
		t.Errorf("Value = %q, want %q", v.Value, str)
	}
	// Odd length: 4 (length prefix) + 3 (string) + 1 (padding) = 8
	if consumed != 8 {
		t.Errorf("consumed = %d, want 8", consumed)
	}
}

func TestLString_GetBytes_RoundTrip(t *testing.T) {
	tests := []string{"", "hi", "hey", "test", "hello world"}
	for _, val := range tests {
		v := lingo.NewLString(val)
		b := v.GetBytes()
		parsed := lingo.FromRawBytes(b, 0)
		str, ok := parsed.(*lingo.LString)
		if !ok {
			t.Fatalf("round-trip for %q: wrong type %T", val, parsed)
		}
		if str.Value != val {
			t.Errorf("round-trip for %q: got %q", val, str.Value)
		}
	}
}

func TestLString_String(t *testing.T) {
	v := lingo.NewLString("hello")
	got := v.String()
	want := `"hello"`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
