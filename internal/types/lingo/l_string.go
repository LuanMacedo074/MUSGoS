package lingo

import (
    "encoding/binary"
    "fmt"
)

type LString struct {
    BaseLValue
    Value string
}

func NewLString(val string) *LString {
    return &LString{
        BaseLValue: BaseLValue{ValueType: VtString},
        Value:      val,
    }
}

func (v *LString) ExtractFromBytes(rawBytes []byte, offset int) int {
    if offset+4 > len(rawBytes) {
        return 0
    }
    
    length := int(binary.BigEndian.Uint32(rawBytes[offset:]))
    if offset+4+length > len(rawBytes) {
        return 0
    }
    
    v.Value = string(rawBytes[offset+4 : offset+4+length])
    return 4 + length
}

func (v *LString) String() string {
    return fmt.Sprintf("\"%s\"", v.Value)
}
