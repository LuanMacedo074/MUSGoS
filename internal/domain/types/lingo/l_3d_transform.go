package lingo

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

type L3dTransform struct {
	BaseLValue
	Matrix [16]float32
}

func NewL3dTransform(matrix [16]float32) *L3dTransform {
	return &L3dTransform{
		BaseLValue: BaseLValue{ValueType: Vt3dTransform},
		Matrix:     matrix,
	}
}

func (v *L3dTransform) ExtractFromBytes(rawBytes []byte, offset int) int {
	if offset+64 > len(rawBytes) {
		return 0
	}
	for i := 0; i < 16; i++ {
		v.Matrix[i] = math.Float32frombits(binary.BigEndian.Uint32(rawBytes[offset+i*4:]))
	}
	return 64
}

func (v *L3dTransform) GetBytes() []byte {
	buf := make([]byte, 66)
	binary.BigEndian.PutUint16(buf[0:], uint16(Vt3dTransform))
	for i := 0; i < 16; i++ {
		binary.BigEndian.PutUint32(buf[2+i*4:], math.Float32bits(v.Matrix[i]))
	}
	return buf
}

func (v *L3dTransform) String() string {
	var parts []string
	for _, f := range v.Matrix {
		parts = append(parts, fmt.Sprintf("%f", f))
	}
	return fmt.Sprintf("transform([%s])", strings.Join(parts, ", "))
}
