package lingo

import (
    "encoding/binary"
    "fmt"
    "math"
)

type LFloat struct {
    BaseLValue
    Value float64
}

func NewLFloat(val float64) *LFloat {
    return &LFloat{
        BaseLValue: BaseLValue{ValueType: VtFloat},
        Value:      val,
    }
}

func (v *LFloat) ExtractFromBytes(rawBytes []byte, offset int) int {
    if offset+8 <= len(rawBytes) {
        bits := binary.BigEndian.Uint64(rawBytes[offset:])
        v.Value = math.Float64frombits(bits)
        return 8
    }
    return 0
}

func (v *LFloat) ToDouble() float64 {
    return v.Value
}

func (v *LFloat) String() string {
    return fmt.Sprintf("%f", v.Value)
}
