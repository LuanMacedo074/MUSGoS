package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLColor_NewAndFields(t *testing.T) {
	v := lingo.NewLColor(255, 128, 0)
	if v.GetType() != lingo.VtColor {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtColor)
	}
	if v.Red != 255 || v.Green != 128 || v.Blue != 0 {
		t.Errorf("RGB = (%d,%d,%d), want (255,128,0)", v.Red, v.Green, v.Blue)
	}
}

func TestLColor_GetBytes_RoundTrip(t *testing.T) {
	v := lingo.NewLColor(10, 20, 30)
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	c, ok := parsed.(*lingo.LColor)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *LColor", parsed)
	}
	if c.Red != 10 || c.Green != 20 || c.Blue != 30 {
		t.Errorf("RGB = (%d,%d,%d), want (10,20,30)", c.Red, c.Green, c.Blue)
	}
}

func TestLColor_String(t *testing.T) {
	v := lingo.NewLColor(255, 0, 128)
	got := v.String()
	if got != "color(255, 0, 128)" {
		t.Errorf("String() = %q, want %q", got, "color(255, 0, 128)")
	}
}
