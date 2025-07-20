package lingo

import (
    "encoding/binary"
    "fmt"
)

type LSymbol struct {
    BaseLValue
    Value string
}

func NewLSymbol(val string) *LSymbol {
    return &LSymbol{
        BaseLValue: BaseLValue{ValueType: VtSymbol},
        Value:      val,
    }
}

func (v *LSymbol) ExtractFromBytes(rawBytes []byte, offset int) int {
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

func (v *LSymbol) String() string {
    return fmt.Sprintf("#%s", v.Value)
}
