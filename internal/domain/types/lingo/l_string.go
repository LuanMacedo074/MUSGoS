package lingo

import (
	"encoding/binary"
	"fmt"
)

type LString struct {
	BaseLValue
	Value string
}

func NewLString(val string) *LString {
	return &LString{
		BaseLValue: BaseLValue{ValueType: VtString},
		Value:      val,
	}
}

func (v *LString) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+4 > len(rawBytes) {
		return 0
	}

	length := int(binary.BigEndian.Uint32(rawBytes[offset:]))
	if offset+4+length > len(rawBytes) {
		return 0
	}

	v.Value = string(rawBytes[offset+4 : offset+4+length])

	paddedLength := length
	if length%2 != 0 {
		paddedLength = length + 1
	}

	return 4 + paddedLength
}

func (v *LString) GetBytes() []byte {
	strBytes := []byte(v.Value)
	length := len(strBytes)
	paddedLength := length
	if length%2 != 0 {
		paddedLength = length + 1
	}
	buf := make([]byte, 2+4+paddedLength)
	binary.BigEndian.PutUint16(buf[0:], uint16(VtString))
	binary.BigEndian.PutUint32(buf[2:], uint32(length))
	copy(buf[6:], strBytes)
	return buf
}

func (v *LString) String() string {
	return fmt.Sprintf("\"%s\"", v.Value)
}
