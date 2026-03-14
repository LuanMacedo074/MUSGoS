package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLPoint_NewAndFields(t *testing.T) {
	v := lingo.NewLPoint(lingo.NewLInteger(10), lingo.NewLInteger(20))
	if v.GetType() != lingo.VtPoint {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtPoint)
	}
	if v.LocH.ToInteger() != 10 {
		t.Errorf("LocH = %d, want 10", v.LocH.ToInteger())
	}
	if v.LocV.ToInteger() != 20 {
		t.Errorf("LocV = %d, want 20", v.LocV.ToInteger())
	}
}

func TestLPoint_GetBytes_RoundTrip(t *testing.T) {
	v := lingo.NewLPoint(lingo.NewLInteger(100), lingo.NewLInteger(200))
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	pt, ok := parsed.(*lingo.LPoint)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *LPoint", parsed)
	}
	if pt.LocH.ToInteger() != 100 {
		t.Errorf("LocH = %d, want 100", pt.LocH.ToInteger())
	}
	if pt.LocV.ToInteger() != 200 {
		t.Errorf("LocV = %d, want 200", pt.LocV.ToInteger())
	}
}

func TestLPoint_GetBytes_RoundTrip_Float(t *testing.T) {
	v := lingo.NewLPoint(lingo.NewLFloat(1.5), lingo.NewLFloat(2.5))
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	pt, ok := parsed.(*lingo.LPoint)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *LPoint", parsed)
	}
	if pt.LocH.ToDouble() != 1.5 {
		t.Errorf("LocH = %f, want 1.5", pt.LocH.ToDouble())
	}
	if pt.LocV.ToDouble() != 2.5 {
		t.Errorf("LocV = %f, want 2.5", pt.LocV.ToDouble())
	}
}

func TestLPoint_String(t *testing.T) {
	v := lingo.NewLPoint(lingo.NewLInteger(10), lingo.NewLInteger(20))
	got := v.String()
	if got != "point(10, 20)" {
		t.Errorf("String() = %q, want %q", got, "point(10, 20)")
	}
}
