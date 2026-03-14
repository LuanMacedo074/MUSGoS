package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLRect_NewAndFields(t *testing.T) {
	v := lingo.NewLRect(lingo.NewLInteger(0), lingo.NewLInteger(0), lingo.NewLInteger(100), lingo.NewLInteger(200))
	if v.GetType() != lingo.VtRect {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtRect)
	}
	if v.Left.ToInteger() != 0 || v.Right.ToInteger() != 100 {
		t.Errorf("unexpected field values")
	}
}

func TestLRect_GetBytes_RoundTrip(t *testing.T) {
	v := lingo.NewLRect(lingo.NewLInteger(10), lingo.NewLInteger(20), lingo.NewLInteger(30), lingo.NewLInteger(40))
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	r, ok := parsed.(*lingo.LRect)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *LRect", parsed)
	}
	if r.Left.ToInteger() != 10 || r.Top.ToInteger() != 20 || r.Right.ToInteger() != 30 || r.Bottom.ToInteger() != 40 {
		t.Errorf("fields = (%d,%d,%d,%d), want (10,20,30,40)", r.Left.ToInteger(), r.Top.ToInteger(), r.Right.ToInteger(), r.Bottom.ToInteger())
	}
}

func TestLRect_String(t *testing.T) {
	v := lingo.NewLRect(lingo.NewLInteger(1), lingo.NewLInteger(2), lingo.NewLInteger(3), lingo.NewLInteger(4))
	got := v.String()
	if got != "rect(1, 2, 3, 4)" {
		t.Errorf("String() = %q, want %q", got, "rect(1, 2, 3, 4)")
	}
}
