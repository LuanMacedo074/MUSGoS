package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestL3dVector_NewAndFields(t *testing.T) {
	v := lingo.NewL3dVector(1.0, 2.0, 3.0)
	if v.GetType() != lingo.Vt3dVector {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.Vt3dVector)
	}
	if v.X != 1.0 || v.Y != 2.0 || v.Z != 3.0 {
		t.Errorf("XYZ = (%f,%f,%f), want (1,2,3)", v.X, v.Y, v.Z)
	}
}

func TestL3dVector_GetBytes_RoundTrip(t *testing.T) {
	v := lingo.NewL3dVector(1.5, -2.5, 3.75)
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	vec, ok := parsed.(*lingo.L3dVector)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *L3dVector", parsed)
	}
	if vec.X != 1.5 || vec.Y != -2.5 || vec.Z != 3.75 {
		t.Errorf("XYZ = (%f,%f,%f), want (1.5,-2.5,3.75)", vec.X, vec.Y, vec.Z)
	}
}

func TestL3dVector_String(t *testing.T) {
	v := lingo.NewL3dVector(1.0, 2.0, 3.0)
	got := v.String()
	if got == "" {
		t.Error("String() returned empty string")
	}
}
