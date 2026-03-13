package lingo_test

import (
	"encoding/binary"
	"math"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLFloat_NewAndValue(t *testing.T) {
	v := lingo.NewLFloat(3.14)
	if v.Value != 3.14 {
		t.Errorf("Value = %f, want 3.14", v.Value)
	}
	if v.GetType() != lingo.VtFloat {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtFloat)
	}
}

func TestLFloat_ExtractFromBytes(t *testing.T) {
	val := 3.14
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(val))

	v := &lingo.LFloat{}
	consumed := v.ExtractFromBytes(buf, 0)
	if consumed != 8 {
		t.Errorf("consumed = %d, want 8", consumed)
	}
	if v.Value != val {
		t.Errorf("Value = %f, want %f", v.Value, val)
	}
}

func TestLFloat_String(t *testing.T) {
	v := lingo.NewLFloat(3.14)
	got := v.String()
	if got == "" {
		t.Error("String() should not be empty")
	}
}

func TestLFloat_ToDouble(t *testing.T) {
	v := lingo.NewLFloat(2.718)
	if v.ToDouble() != 2.718 {
		t.Errorf("ToDouble() = %f, want 2.718", v.ToDouble())
	}
}
