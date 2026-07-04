package lingo_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

// symbolBytes builds a Lingo symbol value: type(2) + len(4) + string (even len,
// no padding needed).
func symbolBytes(s string) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(lingo.VtSymbol))
	l := make([]byte, 4)
	binary.BigEndian.PutUint32(l, uint32(len(s)))
	b = append(b, l...)
	b = append(b, []byte(s)...)
	return b
}

// A truncated proplist — a property with no following value — must leave
// Properties and Values aligned so that GetBytes()/String() do not panic with an
// index-out-of-range (B3). Before the fix, the dangling property was appended
// without its value, giving len(Properties) == len(Values)+1.
func TestLPropList_TruncatedPair_NoPanic(t *testing.T) {
	var raw []byte
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, 1) // claims one pair
	raw = append(raw, count...)
	raw = append(raw, symbolBytes("ab")...) // property, then EOF (no value)

	pl := lingo.NewLPropList()
	pl.ExtractFromBytes(raw, 0)

	if len(pl.Properties) != len(pl.Values) {
		t.Fatalf("slices misaligned: %d properties vs %d values", len(pl.Properties), len(pl.Values))
	}

	// These panicked pre-fix when the slices were mismatched.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("serialize panicked on truncated proplist: %v", r)
		}
	}()
	_ = pl.GetBytes()
	_ = pl.String()
}

// A complete pair must still parse (positive control).
func TestLPropList_CompletePair_Parses(t *testing.T) {
	var raw []byte
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, 1)
	raw = append(raw, count...)
	raw = append(raw, symbolBytes("ab")...) // property

	val := make([]byte, 2) // value: integer type + 4-byte payload
	binary.BigEndian.PutUint16(val, uint16(lingo.VtInteger))
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, 42)
	raw = append(raw, val...)
	raw = append(raw, payload...)

	pl := lingo.NewLPropList()
	pl.ExtractFromBytes(raw, 0)

	if pl.Count() != 1 {
		t.Fatalf("expected 1 pair, got %d", pl.Count())
	}
	if len(pl.Properties) != len(pl.Values) {
		t.Fatalf("slices misaligned: %d vs %d", len(pl.Properties), len(pl.Values))
	}
}
