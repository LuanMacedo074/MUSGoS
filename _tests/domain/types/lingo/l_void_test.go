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

func TestLVoid_GetBytes(t *testing.T) {
	v := lingo.NewLVoid()
	b := v.GetBytes()
	if len(b) != 2 {
		t.Fatalf("len(GetBytes()) = %d, want 2", len(b))
	}
	parsed := lingo.FromRawBytes(b, 0)
	if parsed.GetType() != lingo.VtVoid {
		t.Errorf("round-trip type = %d, want %d", parsed.GetType(), lingo.VtVoid)
	}
}

func TestLVoid_String(t *testing.T) {
	v := lingo.NewLVoid()
	got := v.String()
	if got == "" {
		t.Error("String() should not be empty")
	}
}
