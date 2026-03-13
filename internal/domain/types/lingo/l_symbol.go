package lingo

import (
	"encoding/binary"
)

type LSymbol struct {
	BaseLValue
	Value string
}

func NewLSymbol(val string) *LSymbol {
	return &LSymbol{
		BaseLValue: BaseLValue{ValueType: VtSymbol},
		Value:      val,
	}
}

func (v *LSymbol) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+4 > len(rawBytes) {
		return 0
	}

	length := int(binary.BigEndian.Uint32(rawBytes[offset:]))
	if offset+4+length > len(rawBytes) {
		return 0
	}

	v.Value = string(rawBytes[offset+4 : offset+4+length])

	bytesConsumed := 4 + length

	// Se o símbolo for ímpar, o MUS adiciona um byte de padding
	if length%2 != 0 {
		bytesConsumed++
	}

	return bytesConsumed
}

func (v *LSymbol) GetBytes() []byte {
	symBytes := []byte(v.Value)
	length := len(symBytes)
	paddedLength := length
	if length%2 != 0 {
		paddedLength = length + 1
	}
	buf := make([]byte, 2+4+paddedLength)
	binary.BigEndian.PutUint16(buf[0:], uint16(VtSymbol))
	binary.BigEndian.PutUint32(buf[2:], uint32(length))
	copy(buf[6:], symBytes)
	return buf
}

func (v *LSymbol) String() string {
	return v.Value
}
