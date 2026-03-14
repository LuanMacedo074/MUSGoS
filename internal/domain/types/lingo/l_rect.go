package lingo

import (
	"encoding/binary"
	"fmt"
)

type LRect struct {
	BaseLValue
	Left   LValue
	Top    LValue
	Right  LValue
	Bottom LValue
}

func NewLRect(left, top, right, bottom LValue) *LRect {
	return &LRect{
		BaseLValue: BaseLValue{ValueType: VtRect},
		Left:       left,
		Top:        top,
		Right:      right,
		Bottom:     bottom,
	}
}

func (v *LRect) ExtractFromBytes(rawBytes []byte, offset int) int {
	consumed := 0
	fields := []*LValue{&v.Left, &v.Top, &v.Right, &v.Bottom}
	for _, field := range fields {
		if offset+consumed+2 > len(rawBytes) {
			return 0
		}
		*field = FromRawBytes(rawBytes, offset+consumed)
		c := 2 + (*field).ExtractFromBytes(rawBytes, offset+consumed+2)
		consumed += c
	}
	return consumed
}

func (v *LRect) GetBytes() []byte {
	parts := [][]byte{
		v.Left.GetBytes(),
		v.Top.GetBytes(),
		v.Right.GetBytes(),
		v.Bottom.GetBytes(),
	}
	totalLen := 2
	for _, p := range parts {
		totalLen += len(p)
	}
	buf := make([]byte, 2, totalLen)
	binary.BigEndian.PutUint16(buf[0:], uint16(VtRect))
	for _, p := range parts {
		buf = append(buf, p...)
	}
	return buf
}

func (v *LRect) String() string {
	return fmt.Sprintf("rect(%s, %s, %s, %s)", v.Left.String(), v.Top.String(), v.Right.String(), v.Bottom.String())
}
