package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLDate_NewAndData(t *testing.T) {
	data := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	v := lingo.NewLDate(data)
	if v.GetType() != lingo.VtDate {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtDate)
	}
	if v.Data != data {
		t.Errorf("Data mismatch")
	}
}

func TestLDate_GetBytes_RoundTrip(t *testing.T) {
	data := [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}
	v := lingo.NewLDate(data)
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	d, ok := parsed.(*lingo.LDate)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *LDate", parsed)
	}
	if d.Data != data {
		t.Errorf("Data = %x, want %x", d.Data, data)
	}
}

func TestLDate_ToBytes(t *testing.T) {
	data := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	v := lingo.NewLDate(data)
	got := v.ToBytes()
	if len(got) != 8 {
		t.Errorf("ToBytes() length = %d, want 8", len(got))
	}
}

func TestLDate_String(t *testing.T) {
	data := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	v := lingo.NewLDate(data)
	got := v.String()
	if got == "" {
		t.Error("String() returned empty string")
	}
}
