package smus

import (
    "encoding/binary"
    "errors"
)

type MUSMsgHeaderStringList struct {
    Count   int
    Strings []MUSMsgHeaderString
}

func (m *MUSMsgHeaderStringList) ExtractMUSMsgHeaderStringList(data []byte, offset int) (int, error) {
    if offset+4 > len(data) {
        return 0, errors.New("insufficient data for list count")
    }
    
    count := int(binary.BigEndian.Uint32(data[offset:]))
    m.Count = count
    m.Strings = make([]MUSMsgHeaderString, m.Count)
    
    bytesConsumed := 4
    currentOffset := offset + 4
    
    for i := 0; i < m.Count; i++ {
        str := &m.Strings[i]
        consumed, err := str.ExtractMUSMsgHeaderString(data, currentOffset)
        if err != nil {
            return 0, err
        }
        currentOffset += consumed
        bytesConsumed += consumed
    }
    
    return bytesConsumed, nil
}