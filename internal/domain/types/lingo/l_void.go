package lingo

import "encoding/binary"

type LVoid struct {
	BaseLValue
}

func NewLVoid() *LVoid {
	return &LVoid{BaseLValue{ValueType: VtVoid}}
}

func (v *LVoid) GetBytes() []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(VtVoid))
	return buf
}
