package smus

import (
	"bytes"
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
    if count < 0 {
        return 0, errors.New("invalid negative list count")
    }
    // Each element is at least 4 bytes (its own length prefix), so a count that
    // cannot fit in the remaining bytes is a malformed/hostile wire count. Reject
    // it before allocating to avoid a wire-controlled multi-GB allocation.
    const minElemSize = 4
    if count > (len(data)-offset-4)/minElemSize {
        return 0, errors.New("list count exceeds available data")
    }
    m.Count = count
    m.Strings = make([]MUSMsgHeaderString, 0, count)

    bytesConsumed := 4
    currentOffset := offset + 4

    for i := 0; i < count; i++ {
        var str MUSMsgHeaderString
        consumed, err := str.ExtractMUSMsgHeaderString(data, currentOffset)
        if err != nil {
            return 0, err
        }
        m.Strings = append(m.Strings, str)
        currentOffset += consumed
        bytesConsumed += consumed
    }

    return bytesConsumed, nil
}

func (m *MUSMsgHeaderStringList) WriteBytes(buf *bytes.Buffer) {
	binary.Write(buf, binary.BigEndian, int32(len(m.Strings)))
	for i := range m.Strings {
		m.Strings[i].WriteBytes(buf)
	}
}