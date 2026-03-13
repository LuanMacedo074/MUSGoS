package smus

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type MUSMsgHeaderString struct {
    Length int
    Value  string
}

func (m *MUSMsgHeaderString) ExtractMUSMsgHeaderString(data []byte, offset int) (int, error) {
    if offset+4 > len(data) {
        return 0, errors.New("insufficient data for string length")
    }
    
    length := int(binary.BigEndian.Uint32(data[offset:]))
    m.Length = length
    
    if length < 0 || offset+4+length > len(data) {
        return 0, errors.New("insufficient data for string content")
    }
    
    m.Value = string(data[offset+4 : offset+4+length])
    
    bytesConsumed := 4 + length
    
    // se a string for impar, o MUS adiciona um byte de padding
    if length%2 != 0 {
        bytesConsumed++
    }
    
    return bytesConsumed, nil
}

func (m *MUSMsgHeaderString) WriteBytes(buf *bytes.Buffer) {
	binary.Write(buf, binary.BigEndian, int32(len(m.Value)))
	buf.WriteString(m.Value)
	if len(m.Value)%2 != 0 {
		buf.WriteByte(0x00)
	}
}