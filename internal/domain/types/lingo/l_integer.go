package lingo

import (
    "encoding/binary"
    "fmt"
)

type LInteger struct {
    BaseLValue
    Value int32
}

func NewLInteger(val int32) *LInteger {
    return &LInteger{
        BaseLValue: BaseLValue{ValueType: VtInteger},
        Value:      val,
    }
}

func (v *LInteger) ExtractFromBytes(rawBytes []byte, offset int) int {
    if offset+4 <= len(rawBytes) {
        v.Value = int32(binary.BigEndian.Uint32(rawBytes[offset:]))
        return 4
    }
    return 0
}

func (v *LInteger) ToInteger() int32 {
    return v.Value
}

func (v *LInteger) String() string {
    return fmt.Sprintf("%d", v.Value)
}
