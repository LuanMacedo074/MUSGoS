package lingo_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLInteger_NewAndValue(t *testing.T) {
	v := lingo.NewLInteger(123)
	if v.Value != 123 {
		t.Errorf("Value = %d, want 123", v.Value)
	}
	if v.GetType() != lingo.VtInteger {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtInteger)
	}
}

func TestLInteger_ExtractFromBytes(t *testing.T) {
	tests := []struct {
		name string
		val  int32
	}{
		{"positive", 42},
		{"negative", -100},
		{"zero", 0},
		{"max", 2147483647},
		{"min", -2147483648},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, 4)
			binary.BigEndian.PutUint32(buf, uint32(tt.val))

			v := &lingo.LInteger{}
			consumed := v.ExtractFromBytes(buf, 0)
			if consumed != 4 {
				t.Errorf("consumed = %d, want 4", consumed)
			}
			if v.Value != tt.val {
				t.Errorf("Value = %d, want %d", v.Value, tt.val)
			}
		})
	}
}

func TestLInteger_String(t *testing.T) {
	v := lingo.NewLInteger(42)
	got := v.String()
	if got != "42" {
		t.Errorf("String() = %q, want %q", got, "42")
	}
}

func TestLInteger_ToInteger(t *testing.T) {
	v := lingo.NewLInteger(99)
	if v.ToInteger() != 99 {
		t.Errorf("ToInteger() = %d, want 99", v.ToInteger())
	}
}
