package lingo_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestFromRawBytes_Integer(t *testing.T) {
	var buf []byte
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtInteger))
	buf = append(buf, typeBytes...)
	valBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valBytes, 42)
	buf = append(buf, valBytes...)

	v := lingo.FromRawBytes(buf, 0)
	if v.GetType() != lingo.VtInteger {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtInteger)
	}
	if v.ToInteger() != 42 {
		t.Errorf("ToInteger() = %d, want 42", v.ToInteger())
	}
}

func TestFromRawBytes_String(t *testing.T) {
	var buf []byte
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtString))
	buf = append(buf, typeBytes...)
	str := "hi"
	strLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(strLenBytes, uint32(len(str)))
	buf = append(buf, strLenBytes...)
	buf = append(buf, []byte(str)...)

	v := lingo.FromRawBytes(buf, 0)
	if v.GetType() != lingo.VtString {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtString)
	}
	if v.String() != `"hi"` {
		t.Errorf("String() = %q, want %q", v.String(), `"hi"`)
	}
}

func TestFromRawBytes_Float(t *testing.T) {
	var buf []byte
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtFloat))
	buf = append(buf, typeBytes...)
	floatBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(floatBytes, math.Float64bits(3.14))
	buf = append(buf, floatBytes...)

	v := lingo.FromRawBytes(buf, 0)
	if v.GetType() != lingo.VtFloat {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtFloat)
	}
	if v.ToDouble() != 3.14 {
		t.Errorf("ToDouble() = %f, want 3.14", v.ToDouble())
	}
}

func TestFromRawBytes_Symbol(t *testing.T) {
	var buf []byte
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtSymbol))
	buf = append(buf, typeBytes...)
	sym := "test"
	strLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(strLenBytes, uint32(len(sym)))
	buf = append(buf, strLenBytes...)
	buf = append(buf, []byte(sym)...)

	v := lingo.FromRawBytes(buf, 0)
	if v.GetType() != lingo.VtSymbol {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtSymbol)
	}
	if v.String() != "test" {
		t.Errorf("String() = %q, want %q", v.String(), "test")
	}
}

func TestFromRawBytes_List(t *testing.T) {
	var buf []byte
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtList))
	buf = append(buf, typeBytes...)

	// count = 1
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, 1)
	buf = append(buf, countBytes...)

	// element: integer 99
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtInteger))
	buf = append(buf, typeBytes...)
	valBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valBytes, 99)
	buf = append(buf, valBytes...)

	v := lingo.FromRawBytes(buf, 0)
	if v.GetType() != lingo.VtList {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtList)
	}
}

func TestFromRawBytes_PropList(t *testing.T) {
	var buf []byte
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtPropList))
	buf = append(buf, typeBytes...)

	// count = 1
	countBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(countBytes, 1)
	buf = append(buf, countBytes...)

	// property: symbol "k"
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtSymbol))
	buf = append(buf, typeBytes...)
	strLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(strLenBytes, 1)
	buf = append(buf, strLenBytes...)
	buf = append(buf, 'k')
	buf = append(buf, 0x00) // padding for odd-length symbol

	// value: integer 5
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtInteger))
	buf = append(buf, typeBytes...)
	valBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valBytes, 5)
	buf = append(buf, valBytes...)

	v := lingo.FromRawBytes(buf, 0)
	if v.GetType() != lingo.VtPropList {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtPropList)
	}
}

func TestFromRawBytes_Void(t *testing.T) {
	var buf []byte
	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, uint16(lingo.VtVoid))
	buf = append(buf, typeBytes...)

	v := lingo.FromRawBytes(buf, 0)
	if v.GetType() != lingo.VtVoid {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtVoid)
	}
}

func TestGetLValue_Int(t *testing.T) {
	v := lingo.GetLValue(42)
	if v.GetType() != lingo.VtInteger {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtInteger)
	}
	if v.ToInteger() != 42 {
		t.Errorf("ToInteger() = %d, want 42", v.ToInteger())
	}
}

func TestGetLValue_String(t *testing.T) {
	v := lingo.GetLValue("hello")
	if v.GetType() != lingo.VtString {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtString)
	}
}

func TestGetLValue_Float64(t *testing.T) {
	v := lingo.GetLValue(3.14)
	if v.GetType() != lingo.VtFloat {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtFloat)
	}
	if v.ToDouble() != 3.14 {
		t.Errorf("ToDouble() = %f, want 3.14", v.ToDouble())
	}
}

func TestGetLValue_Bytes(t *testing.T) {
	data := []byte{1, 2, 3}
	v := lingo.GetLValue(data)
	if v.GetType() != lingo.VtMedia {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtMedia)
	}
	if !bytes.Equal(v.ToBytes(), data) {
		t.Errorf("ToBytes() = %v, want %v", v.ToBytes(), data)
	}
}

func TestGetLValue_Nil(t *testing.T) {
	v := lingo.GetLValue(nil)
	if v.GetType() != lingo.VtVoid {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtVoid)
	}
}
