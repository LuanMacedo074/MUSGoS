package lingo_test

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

// buildPropListBytes builds raw bytes for a proplist with symbol keys and integer values.
func buildPropListBytes(pairs [][2]interface{}) []byte {
	var buf []byte

	// count
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, uint32(len(pairs)))
	buf = append(buf, countBytes...)

	typeBytes := make([]byte, 2)

	for _, pair := range pairs {
		propName := pair[0].(string)
		val := pair[1].(int32)

		// Property: symbol type + length + string
		binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtSymbol))
		buf = append(buf, typeBytes...)
		strLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(strLenBytes, uint32(len(propName)))
		buf = append(buf, strLenBytes...)
		buf = append(buf, []byte(propName)...)
		if len(propName)%2 != 0 {
			buf = append(buf, 0x00) // padding
		}

		// Value: integer type + 4 bytes
		binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtInteger))
		buf = append(buf, typeBytes...)
		valBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(valBytes, uint32(val))
		buf = append(buf, valBytes...)
	}

	return buf
}

func TestLPropList_ExtractFromBytes(t *testing.T) {
	pairs := [][2]interface{}{
		{"age", int32(25)},
		{"score", int32(100)},
	}
	buf := buildPropListBytes(pairs)

	v := lingo.NewLPropList()
	consumed := v.ExtractFromBytes(buf, 0)
	if consumed == 0 {
		t.Fatal("ExtractFromBytes returned 0")
	}
	if v.Count() != 2 {
		t.Errorf("Count() = %d, want 2", v.Count())
	}
}

func TestLPropList_GetElement(t *testing.T) {
	pairs := [][2]interface{}{
		{"name", int32(42)},
		{"id", int32(7)},
	}
	buf := buildPropListBytes(pairs)

	v := lingo.NewLPropList()
	v.ExtractFromBytes(buf, 0)

	// Found
	val, err := v.GetElement("name")
	if err != nil {
		t.Fatalf("GetElement('name') error: %v", err)
	}
	if val.ToInteger() != 42 {
		t.Errorf("GetElement('name').ToInteger() = %d, want 42", val.ToInteger())
	}

	// Not found
	_, err = v.GetElement("missing")
	if err == nil {
		t.Error("GetElement('missing') should return error")
	}
}

func TestLPropList_GetElementAt(t *testing.T) {
	pairs := [][2]interface{}{
		{"aa", int32(10)},
		{"bb", int32(20)},
	}
	buf := buildPropListBytes(pairs)

	v := lingo.NewLPropList()
	v.ExtractFromBytes(buf, 0)

	// Valid index
	val := v.GetElementAt(0)
	if val.ToInteger() != 10 {
		t.Errorf("GetElementAt(0).ToInteger() = %d, want 10", val.ToInteger())
	}

	val = v.GetElementAt(1)
	if val.ToInteger() != 20 {
		t.Errorf("GetElementAt(1).ToInteger() = %d, want 20", val.ToInteger())
	}

	// Out of range
	val = v.GetElementAt(99)
	if val.GetType() != lingo.VtVoid {
		t.Errorf("GetElementAt(99).GetType() = %d, want VtVoid(%d)", val.GetType(), lingo.VtVoid)
	}

	val = v.GetElementAt(-1)
	if val.GetType() != lingo.VtVoid {
		t.Errorf("GetElementAt(-1).GetType() = %d, want VtVoid(%d)", val.GetType(), lingo.VtVoid)
	}
}

func TestLPropList_GetPropAt(t *testing.T) {
	pairs := [][2]interface{}{
		{"xx", int32(1)},
	}
	buf := buildPropListBytes(pairs)

	v := lingo.NewLPropList()
	v.ExtractFromBytes(buf, 0)

	prop := v.GetPropAt(0)
	if prop.String() != "xx" {
		t.Errorf("GetPropAt(0).String() = %q, want %q", prop.String(), "xx")
	}

	// Out of range
	prop = v.GetPropAt(5)
	if prop.GetType() != lingo.VtVoid {
		t.Errorf("GetPropAt(5).GetType() = %d, want VtVoid", prop.GetType())
	}
}

func TestLPropList_Count(t *testing.T) {
	v := lingo.NewLPropList()
	if v.Count() != 0 {
		t.Errorf("Count() on empty = %d, want 0", v.Count())
	}

	pairs := [][2]interface{}{
		{"a", int32(1)},
		{"b", int32(2)},
		{"c", int32(3)},
	}
	buf := buildPropListBytes(pairs)
	v = lingo.NewLPropList()
	v.ExtractFromBytes(buf, 0)
	if v.Count() != 3 {
		t.Errorf("Count() = %d, want 3", v.Count())
	}
}

func TestLPropList_GetBytes_RoundTrip(t *testing.T) {
	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("name"), lingo.NewLString("test"))
	plist.AddElement(lingo.NewLSymbol("value"), lingo.NewLInteger(42))

	b := plist.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	pl, ok := parsed.(*lingo.LPropList)
	if !ok {
		t.Fatalf("wrong type %T", parsed)
	}
	if pl.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", pl.Count())
	}
	val, err := pl.GetElement("name")
	if err != nil {
		t.Fatalf("GetElement('name') error: %v", err)
	}
	if str, ok := val.(*lingo.LString); !ok || str.Value != "test" {
		t.Errorf("name value = %v, want 'test'", val)
	}
}

func TestLPropList_GetBytes_Structure(t *testing.T) {
	pairs := [][2]interface{}{
		{"key1", int32(100)},
		{"key2", int32(200)},
	}
	buf := buildPropListBytes(pairs)

	v := lingo.NewLPropList()
	v.ExtractFromBytes(buf, 0)

	serialized := v.GetBytes()

	// GetBytes should produce non-empty output
	if len(serialized) == 0 {
		t.Fatal("GetBytes() returned empty")
	}

	// First 2 bytes should be VtPropList type
	gotType := int16(binary.BigEndian.Uint16(serialized[0:]))
	if gotType != lingo.VtPropList {
		t.Errorf("type prefix = %d, want %d", gotType, lingo.VtPropList)
	}

	// Next 4 bytes should be count=2
	gotCount := int32(binary.BigEndian.Uint32(serialized[2:]))
	if gotCount != 2 {
		t.Errorf("count = %d, want 2", gotCount)
	}
}

func TestLPropList_String(t *testing.T) {
	pairs := [][2]interface{}{
		{"x", int32(1)},
	}
	buf := buildPropListBytes(pairs)

	v := lingo.NewLPropList()
	v.ExtractFromBytes(buf, 0)

	got := v.String()
	if got == "" || got == "[]" {
		t.Errorf("String() = %q, want non-empty proplist string", got)
	}
	// Should contain the property name and value
	if got != "[x: 1]" {
		t.Errorf("String() = %q, want %q", got, "[x: 1]")
	}
}
