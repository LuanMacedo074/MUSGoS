package lingo

import (
	"encoding/binary"
	"fmt"
)

const (
	VtVoid        int16 = 0
	VtInteger     int16 = 1
	VtSymbol      int16 = 2
	VtString      int16 = 3
	VtPicture     int16 = 5
	VtFloat       int16 = 6
	VtList        int16 = 7
	VtPoint       int16 = 8
	VtRect        int16 = 9
	VtPropList    int16 = 10
	VtColor       int16 = 18
	VtDate        int16 = 19
	VtMedia       int16 = 20
	Vt3dVector    int16 = 22
	Vt3dTransform int16 = 23
)

type LValue interface {
	GetType() int16
	ExtractFromBytes(rawBytes []byte, offset int) int
	GetBytes() []byte
	String() string
	ToInteger() int32
	ToDouble() float64
	ToBytes() []byte
}

type BaseLValue struct {
	ValueType int16
}

func (v *BaseLValue) GetType() int16 {
	return v.ValueType
}

func (v *BaseLValue) ExtractFromBytes(rawBytes []byte, offset int) int {
	return 0
}

func (v *BaseLValue) GetBytes() []byte {
	return []byte{}
}

func (v *BaseLValue) String() string {
	return fmt.Sprintf("{LValue type %d}", v.ValueType)
}

func (v *BaseLValue) ToInteger() int32 {
	return 0
}

func (v *BaseLValue) ToDouble() float64 {
	return 0.0
}

func (v *BaseLValue) ToBytes() []byte {
	return []byte{}
}

func FromRawBytes(rawBytes []byte, offset int) LValue {
	if offset+2 > len(rawBytes) {
		return NewLVoid()
	}

	elemType := int16(binary.BigEndian.Uint16(rawBytes[offset:]))

	var newVal LValue
	switch elemType {
	case VtVoid:
		newVal = NewLVoid()
	case VtInteger:
		newVal = &LInteger{BaseLValue: BaseLValue{ValueType: VtInteger}}
	case VtSymbol:
		newVal = &LSymbol{BaseLValue: BaseLValue{ValueType: VtSymbol}}
	case VtString:
		newVal = &LString{BaseLValue: BaseLValue{ValueType: VtString}}
	case VtFloat:
		newVal = &LFloat{BaseLValue: BaseLValue{ValueType: VtFloat}}
	case VtList:
		newVal = &LList{BaseLValue: BaseLValue{ValueType: VtList}}
	case VtPicture:
		newVal = &LPicture{BaseLValue: BaseLValue{ValueType: VtPicture}}
	case VtMedia:
		newVal = &LMedia{BaseLValue: BaseLValue{ValueType: VtMedia}}
	case VtPoint:
		newVal = &LPoint{BaseLValue: BaseLValue{ValueType: VtPoint}}
	case VtRect:
		newVal = &LRect{BaseLValue: BaseLValue{ValueType: VtRect}}
	case VtPropList:
		newVal = &LPropList{BaseLValue: BaseLValue{ValueType: VtPropList}}
	case VtColor:
		newVal = &LColor{BaseLValue: BaseLValue{ValueType: VtColor}}
	case VtDate:
		newVal = &LDate{BaseLValue: BaseLValue{ValueType: VtDate}}
	case Vt3dVector:
		newVal = &L3dVector{BaseLValue: BaseLValue{ValueType: Vt3dVector}}
	case Vt3dTransform:
		newVal = &L3dTransform{BaseLValue: BaseLValue{ValueType: Vt3dTransform}}
	default:
		newVal = NewLVoid()
	}

	newVal.ExtractFromBytes(rawBytes, offset+2)
	return newVal
}

// StringValue extracts the raw string from an LValue.
// If the value is an LString, returns its Value directly (without quotes).
// Otherwise falls back to the general String() representation.
func StringValue(v LValue) string {
	if s, ok := v.(*LString); ok {
		return s.Value
	}
	return v.String()
}

// ExtractString extracts a string from an LValue that is expected to carry
// a string argument. It accepts LString, LSymbol, or the first element of
// an LList. Returns an error when the value cannot be interpreted as a string.
func ExtractString(v LValue) (string, error) {
	switch t := v.(type) {
	case *LString:
		return t.Value, nil
	case *LSymbol:
		return t.Value, nil
	case *LList:
		if len(t.Values) > 0 {
			return ExtractString(t.Values[0])
		}
		return "", fmt.Errorf("empty list: cannot extract string")
	default:
		return "", fmt.Errorf("cannot extract string from %T", v)
	}
}

func GetLValue(val interface{}) LValue {
	switch v := val.(type) {
	case int:
		return NewLInteger(int32(v))
	case int32:
		return NewLInteger(v)
	case string:
		return NewLString(v)
	case float64:
		return NewLFloat(v)
	case float32:
		return NewLFloat(float64(v))
	case []byte:
		return NewLMedia(v)
	default:
		return NewLVoid()
	}
}
