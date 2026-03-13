package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLVoid_New(t *testing.T) {
	v := lingo.NewLVoid()
	if v.GetType() != lingo.VtVoid {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtVoid)
	}
}

func TestLVoid_String(t *testing.T) {
	v := lingo.NewLVoid()
	got := v.String()
	if got == "" {
		t.Error("String() should not be empty")
	}
}
