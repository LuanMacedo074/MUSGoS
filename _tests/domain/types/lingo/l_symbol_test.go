package lingo_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLSymbol_NewAndValue(t *testing.T) {
	v := lingo.NewLSymbol("foo")
	if v.Value != "foo" {
		t.Errorf("Value = %q, want %q", v.Value, "foo")
	}
	if v.GetType() != lingo.VtSymbol {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtSymbol)
	}
}

func TestLSymbol_ExtractFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		wantSize int
	}{
		{"even", "ab", 6},    // 4 + 2
		{"odd", "abc", 8},    // 4 + 3 + 1 padding
		{"single", "x", 6},   // 4 + 1 + 1 padding
		{"four", "test", 8},  // 4 + 4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, 4+len(tt.val)+1) // extra byte for potential padding
			binary.BigEndian.PutUint32(buf[0:], uint32(len(tt.val)))
			copy(buf[4:], tt.val)

			v := &lingo.LSymbol{}
			consumed := v.ExtractFromBytes(buf, 0)
			if v.Value != tt.val {
				t.Errorf("Value = %q, want %q", v.Value, tt.val)
			}
			if consumed != tt.wantSize {
				t.Errorf("consumed = %d, want %d", consumed, tt.wantSize)
			}
		})
	}
}

func TestLSymbol_String(t *testing.T) {
	v := lingo.NewLSymbol("mySymbol")
	got := v.String()
	if got != "mySymbol" {
		t.Errorf("String() = %q, want %q", got, "mySymbol")
	}
}
