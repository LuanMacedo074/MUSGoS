package lingo

import (
	"encoding/binary"
	"fmt"
)

type LColor struct {
	BaseLValue
	Red   uint8
	Green uint8
	Blue  uint8
}

func NewLColor(r, g, b uint8) *LColor {
	return &LColor{
		BaseLValue: BaseLValue{ValueType: VtColor},
		Red:        r,
		Green:      g,
		Blue:       b,
	}
}

func (v *LColor) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+4 > len(rawBytes) {
		return 0
	}
	v.Red = rawBytes[offset]
	v.Green = rawBytes[offset+1]
	v.Blue = rawBytes[offset+2]
	// rawBytes[offset+3] is padding
	return 4
}

func (v *LColor) GetBytes() []byte {
	buf := make([]byte, 6)
	binary.BigEndian.PutUint16(buf[0:], uint16(VtColor))
	buf[2] = v.Red
	buf[3] = v.Green
	buf[4] = v.Blue
	buf[5] = 0 // padding
	return buf
}

func (v *LColor) String() string {
	return fmt.Sprintf("color(%d, %d, %d)", v.Red, v.Green, v.Blue)
}
