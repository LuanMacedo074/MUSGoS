package lingo_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

// A wire count larger than the buffer could hold must not pre-allocate a giant
// slice; ExtractFromBytes should bail (return 0) instead (pre-auth DoS, B1).
func TestLList_HugeCount_NoAlloc(t *testing.T) {
	raw := make([]byte, 8)
	binary.BigEndian.PutUint32(raw[0:], 0x0FFFFFFF) // ~268M elements claimed

	list := lingo.NewLList()
	n := list.ExtractFromBytes(raw, 0)
	if n != 0 {
		t.Errorf("expected 0 consumed on bogus count, got %d", n)
	}
	if len(list.Values) != 0 {
		t.Errorf("expected no values allocated, got %d", len(list.Values))
	}
}
