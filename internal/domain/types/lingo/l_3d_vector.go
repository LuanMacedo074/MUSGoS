package lingo

import (
	"encoding/binary"
	"fmt"
	"math"
)

type L3dVector struct {
	BaseLValue
	X float32
	Y float32
	Z float32
}

func NewL3dVector(x, y, z float32) *L3dVector {
	return &L3dVector{
		BaseLValue: BaseLValue{ValueType: Vt3dVector},
		X:          x,
		Y:          y,
		Z:          z,
	}
}

func (v *L3dVector) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+12 > len(rawBytes) {
		return 0
	}
	v.X = math.Float32frombits(binary.BigEndian.Uint32(rawBytes[offset:]))
	v.Y = math.Float32frombits(binary.BigEndian.Uint32(rawBytes[offset+4:]))
	v.Z = math.Float32frombits(binary.BigEndian.Uint32(rawBytes[offset+8:]))
	return 12
}

func (v *L3dVector) GetBytes() []byte {
	buf := make([]byte, 14)
	binary.BigEndian.PutUint16(buf[0:], uint16(Vt3dVector))
	binary.BigEndian.PutUint32(buf[2:], math.Float32bits(v.X))
	binary.BigEndian.PutUint32(buf[6:], math.Float32bits(v.Y))
	binary.BigEndian.PutUint32(buf[10:], math.Float32bits(v.Z))
	return buf
}

func (v *L3dVector) String() string {
	return fmt.Sprintf("vector(%f, %f, %f)", v.X, v.Y, v.Z)
}
