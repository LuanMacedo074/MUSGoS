package lingo

import (
	"encoding/binary"
	"fmt"
)

// LDate represents a Lingo date value. The 8-byte data format is
// Shockwave/Director-specific and not publicly documented.
type LDate struct {
	BaseLValue
	Data [8]byte
}

func NewLDate(data [8]byte) *LDate {
	return &LDate{
		BaseLValue: BaseLValue{ValueType: VtDate},
		Data:       data,
	}
}

func (v *LDate) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+8 > len(rawBytes) {
		return 0
	}
	copy(v.Data[:], rawBytes[offset:offset+8])
	return 8
}

func (v *LDate) GetBytes() []byte {
	buf := make([]byte, 10)
	binary.BigEndian.PutUint16(buf[0:], uint16(VtDate))
	copy(buf[2:], v.Data[:])
	return buf
}

func (v *LDate) ToBytes() []byte {
	return v.Data[:]
}

func (v *LDate) String() string {
	return fmt.Sprintf("<date %x>", v.Data)
}
