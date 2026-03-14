package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestL3dTransform_NewAndMatrix(t *testing.T) {
	var matrix [16]float32
	for i := range matrix {
		matrix[i] = float32(i)
	}
	v := lingo.NewL3dTransform(matrix)
	if v.GetType() != lingo.Vt3dTransform {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.Vt3dTransform)
	}
	for i, f := range v.Matrix {
		if f != float32(i) {
			t.Errorf("Matrix[%d] = %f, want %f", i, f, float32(i))
		}
	}
}

func TestL3dTransform_GetBytes_RoundTrip(t *testing.T) {
	var matrix [16]float32
	matrix[0] = 1.0
	matrix[5] = 1.0
	matrix[10] = 1.0
	matrix[15] = 1.0
	v := lingo.NewL3dTransform(matrix)
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	tr, ok := parsed.(*lingo.L3dTransform)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *L3dTransform", parsed)
	}
	for i := 0; i < 16; i++ {
		if tr.Matrix[i] != matrix[i] {
			t.Errorf("Matrix[%d] = %f, want %f", i, tr.Matrix[i], matrix[i])
		}
	}
}

func TestL3dTransform_String(t *testing.T) {
	var matrix [16]float32
	v := lingo.NewL3dTransform(matrix)
	got := v.String()
	if got == "" {
		t.Error("String() returned empty string")
	}
}
