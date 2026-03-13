package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestCodecRoundTrip_Void(t *testing.T) {
	original := lingo.NewLVoid()
	data, err := lingo.MarshalLValue(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := lingo.UnmarshalLValue(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GetType() != lingo.VtVoid {
		t.Errorf("expected void, got type %d", got.GetType())
	}
}

func TestCodecRoundTrip_Integer(t *testing.T) {
	original := lingo.NewLInteger(42)
	data, err := lingo.MarshalLValue(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := lingo.UnmarshalLValue(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GetType() != lingo.VtInteger {
		t.Errorf("expected integer type, got %d", got.GetType())
	}
	if got.ToInteger() != 42 {
		t.Errorf("expected 42, got %d", got.ToInteger())
	}
}

func TestCodecRoundTrip_NegativeInteger(t *testing.T) {
	original := lingo.NewLInteger(-999)
	data, err := lingo.MarshalLValue(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := lingo.UnmarshalLValue(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ToInteger() != -999 {
		t.Errorf("expected -999, got %d", got.ToInteger())
	}
}

func TestCodecRoundTrip_Float(t *testing.T) {
	original := lingo.NewLFloat(3.14)
	data, err := lingo.MarshalLValue(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := lingo.UnmarshalLValue(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GetType() != lingo.VtFloat {
		t.Errorf("expected float type, got %d", got.GetType())
	}
	if got.ToDouble() != 3.14 {
		t.Errorf("expected 3.14, got %f", got.ToDouble())
	}
}

func TestCodecRoundTrip_String(t *testing.T) {
	original := lingo.NewLString("hello world")
	data, err := lingo.MarshalLValue(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := lingo.UnmarshalLValue(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GetType() != lingo.VtString {
		t.Errorf("expected string type, got %d", got.GetType())
	}
	ls, ok := got.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", got)
	}
	if ls.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", ls.Value)
	}
}

func TestCodecRoundTrip_Symbol(t *testing.T) {
	original := lingo.NewLSymbol("mySymbol")
	data, err := lingo.MarshalLValue(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := lingo.UnmarshalLValue(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GetType() != lingo.VtSymbol {
		t.Errorf("expected symbol type, got %d", got.GetType())
	}
	ls, ok := got.(*lingo.LSymbol)
	if !ok {
		t.Fatalf("expected *LSymbol, got %T", got)
	}
	if ls.Value != "mySymbol" {
		t.Errorf("expected 'mySymbol', got %q", ls.Value)
	}
}

func TestCodecRoundTrip_EmptyString(t *testing.T) {
	original := lingo.NewLString("")
	data, err := lingo.MarshalLValue(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := lingo.UnmarshalLValue(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ls, ok := got.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", got)
	}
	if ls.Value != "" {
		t.Errorf("expected empty string, got %q", ls.Value)
	}
}

func TestCodecUnmarshal_InvalidJSON(t *testing.T) {
	_, err := lingo.UnmarshalLValue([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
