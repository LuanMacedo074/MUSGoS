package lingo

import (
	"encoding/binary"
	"fmt"
)

type LPicture struct {
	BaseLValue
	Data []byte
}

func NewLPicture(data []byte) *LPicture {
	return &LPicture{
		BaseLValue: BaseLValue{ValueType: VtPicture},
		Data:       data,
	}
}

func (v *LPicture) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+4 > len(rawBytes) {
		return 0
	}
	length := int(binary.BigEndian.Uint32(rawBytes[offset:]))
	if offset+4+length > len(rawBytes) {
		return 0
	}
	v.Data = make([]byte, length)
	copy(v.Data, rawBytes[offset+4:offset+4+length])
	return 4 + length
}

func (v *LPicture) GetBytes() []byte {
	buf := make([]byte, 2+4+len(v.Data))
	binary.BigEndian.PutUint16(buf[0:], uint16(VtPicture))
	binary.BigEndian.PutUint32(buf[2:], uint32(len(v.Data)))
	copy(buf[6:], v.Data)
	return buf
}

func (v *LPicture) ToBytes() []byte {
	return v.Data
}

func (v *LPicture) String() string {
	return fmt.Sprintf("<picture %d bytes>", len(v.Data))
}
