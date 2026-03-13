package lingo

import (
    "encoding/binary"
)

type LList struct {
    BaseLValue
    Values []LValue
}

func NewLList() *LList {
    return &LList{
        BaseLValue: BaseLValue{ValueType: VtList},
        Values:     []LValue{},
    }
}

func (v *LList) ExtractFromBytes(rawBytes []byte, offset int) int {
    if offset+4 > len(rawBytes) {
        return 0
    }
    
    count := int(binary.BigEndian.Uint32(rawBytes[offset:]))
    v.Values = make([]LValue, count)
    
    currentOffset := offset + 4
    
    for i := 0; i < count; i++ {
        elem := FromRawBytes(rawBytes, currentOffset)
        if elem == nil {
            return 0
        }
        v.Values[i] = elem
        consumed := elem.ExtractFromBytes(rawBytes, currentOffset+2)
        currentOffset += 2 + consumed
    }
    
    return currentOffset - offset
}
