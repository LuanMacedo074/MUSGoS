package lingo

import (
	"encoding/binary"
	"fmt"
)

type LPoint struct {
	BaseLValue
	LocH LValue
	LocV LValue
}

func NewLPoint(locH, locV LValue) *LPoint {
	return &LPoint{
		BaseLValue: BaseLValue{ValueType: VtPoint},
		LocH:       locH,
		LocV:       locV,
	}
}

func (v *LPoint) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+2 > len(rawBytes) {
		return 0
	}
	v.LocH = FromRawBytes(rawBytes, offset)
	consumedH := 2 + v.LocH.ExtractFromBytes(rawBytes, offset+2)

	if offset+consumedH+2 > len(rawBytes) {
		return 0
	}
	v.LocV = FromRawBytes(rawBytes, offset+consumedH)
	consumedV := 2 + v.LocV.ExtractFromBytes(rawBytes, offset+consumedH+2)

	return consumedH + consumedV
}

func (v *LPoint) GetBytes() []byte {
	hBytes := v.LocH.GetBytes()
	vBytes := v.LocV.GetBytes()
	buf := make([]byte, 2+len(hBytes)+len(vBytes))
	binary.BigEndian.PutUint16(buf[0:], uint16(VtPoint))
	copy(buf[2:], hBytes)
	copy(buf[2+len(hBytes):], vBytes)
	return buf
}

func (v *LPoint) String() string {
	return fmt.Sprintf("point(%s, %s)", v.LocH.String(), v.LocV.String())
}
