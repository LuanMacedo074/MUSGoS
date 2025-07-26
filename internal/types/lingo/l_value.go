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
	case VtMedia:
		newVal = &LMedia{BaseLValue: BaseLValue{ValueType: VtMedia}}
	// TODO: outros tipos
	// - VtPicture (5)
	// - VtPoint (8)
	// - VtRect (9)
	case VtPropList:
		fmt.Print("Gerou uma proplist")
	// - VtColor (18)
	// - VtDate (19)
	// - Vt3dVector (22)
	// - Vt3dTransform (23)
	default:
		newVal = NewLVoid()
	}

	newVal.ExtractFromBytes(rawBytes, offset+2)
	return newVal
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
