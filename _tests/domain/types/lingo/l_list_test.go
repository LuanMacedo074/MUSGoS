package lingo_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLList_ExtractFromBytes_Empty(t *testing.T) {
	// count=0
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 0)

	v := lingo.NewLList()
	consumed := v.ExtractFromBytes(buf, 0)
	if consumed != 4 {
		t.Errorf("consumed = %d, want 4", consumed)
	}
	if len(v.Values) != 0 {
		t.Errorf("len(Values) = %d, want 0", len(v.Values))
	}
}

func TestLList_GetBytes_RoundTrip(t *testing.T) {
	list := lingo.NewLList()
	list.Values = []lingo.LValue{
		lingo.NewLInteger(42),
		lingo.NewLString("hello"),
		lingo.NewLFloat(3.14),
	}
	b := list.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	l, ok := parsed.(*lingo.LList)
	if !ok {
		t.Fatalf("wrong type %T", parsed)
	}
	if len(l.Values) != 3 {
		t.Fatalf("len(Values) = %d, want 3", len(l.Values))
	}
	if l.Values[0].ToInteger() != 42 {
		t.Errorf("Values[0] = %d, want 42", l.Values[0].ToInteger())
	}
	if l.Values[2].ToDouble() != 3.14 {
		t.Errorf("Values[2] = %f, want 3.14", l.Values[2].ToDouble())
	}
}

func TestLList_ExtractFromBytes_Mixed(t *testing.T) {
	// List with 2 elements: integer(42) + string("hi")
	var buf []byte

	// count = 2
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, 2)
	buf = append(buf, countBytes...)

	// Element 1: type=VtInteger(1), value=42
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtInteger))
	buf = append(buf, typeBytes...)
	valBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valBytes, 42)
	buf = append(buf, valBytes...)

	// Element 2: type=VtString(3), length=2, value="hi"
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtString))
	buf = append(buf, typeBytes...)
	strLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(strLenBytes, 2)
	buf = append(buf, strLenBytes...)
	buf = append(buf, []byte("hi")...)

	v := lingo.NewLList()
	consumed := v.ExtractFromBytes(buf, 0)
	if consumed == 0 {
		t.Fatal("ExtractFromBytes returned 0")
	}
	if len(v.Values) != 2 {
		t.Fatalf("len(Values) = %d, want 2", len(v.Values))
	}
	if v.Values[0].GetType() != lingo.VtInteger {
		t.Errorf("Values[0].GetType() = %d, want %d", v.Values[0].GetType(), lingo.VtInteger)
	}
	if v.Values[0].ToInteger() != 42 {
		t.Errorf("Values[0].ToInteger() = %d, want 42", v.Values[0].ToInteger())
	}
	if v.Values[1].GetType() != lingo.VtString {
		t.Errorf("Values[1].GetType() = %d, want %d", v.Values[1].GetType(), lingo.VtString)
	}
}
