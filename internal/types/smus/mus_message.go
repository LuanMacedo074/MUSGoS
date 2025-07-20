package smus

import (
    "encoding/binary"
    "errors"
    "fmt"
    
    "fsos-server/internal/types/lingo"
)

// header padrão do MUS, vem em todas as messagem
var MUSHeader = []byte{0x72, 0x00}

type MUSMessage struct {
    ContentSize int32
    ErrCode     int32
    TimeStamp   int32
    Subject     MUSMsgHeaderString
    SenderID    MUSMsgHeaderString
    RecptID     MUSMsgHeaderStringList
    RawContents []byte
    MsgContent  lingo.LValue
}

func ParseMUSMessage(rawmsg []byte) (*MUSMessage, error) {
    if len(rawmsg) < 14 {
        return nil, errors.New("message too short")
    }
    
    // verifica header mus
    if rawmsg[0] != MUSHeader[0] || rawmsg[1] != MUSHeader[1] {
        return nil, fmt.Errorf("invalid MUS header: expected %X, got %X", MUSHeader, rawmsg[:2])
    }

    // é importante pular os 6 primeiros bytes que é o header padrão + o tamanho da mensagem, ai começamos a parsear a partir do offset 6
    msg := &MUSMessage{}
    readPtr := 2
    
    // le tamanho do conteudo
    msg.ContentSize = int32(binary.BigEndian.Uint32(rawmsg[readPtr:]))
    readPtr += 4
    
    if len(rawmsg) < int(6 + msg.ContentSize) {
        return nil, fmt.Errorf("message truncated: expected %d bytes, got %d", 6 + msg.ContentSize, len(rawmsg))
    }
    
    // campos obrigatorios do header
    msg.ErrCode = int32(binary.BigEndian.Uint32(rawmsg[readPtr:]))
    readPtr += 4
    
    msg.TimeStamp = int32(binary.BigEndian.Uint32(rawmsg[readPtr:]))
    readPtr += 4
    
    // strings do header usam metodos proprios de extracao
    consumed, err := msg.Subject.ExtractMUSMsgHeaderString(rawmsg, readPtr)
    if err != nil {
        return nil, fmt.Errorf("failed to extract subject: %w", err)
    }
    readPtr += consumed
    
    consumed, err = msg.SenderID.ExtractMUSMsgHeaderString(rawmsg, readPtr)
    if err != nil {
        return nil, fmt.Errorf("failed to extract sender ID: %w", err)
    }
    readPtr += consumed
    
    consumed, err = msg.RecptID.ExtractMUSMsgHeaderStringList(rawmsg, readPtr)
    if err != nil {
        return nil, fmt.Errorf("failed to extract recipient list: %w", err)
    }
    readPtr += consumed
    
    // conteudo da mensagem
    if readPtr < len(rawmsg) {
        msg.RawContents = make([]byte, len(rawmsg)-readPtr)
        copy(msg.RawContents, rawmsg[readPtr:])
        msg.MsgContent = lingo.FromRawBytes(msg.RawContents, 0)
    } else {
        msg.RawContents = []byte{}
        msg.MsgContent = lingo.NewLVoid()
    }
    
    return msg, nil
}

func (msg *MUSMessage) String() string {
    result := "MUS Message:\n"
    result += fmt.Sprintf("  Content Size: %d bytes\n", msg.ContentSize)
    result += fmt.Sprintf("  Error Code: %d (0x%08X)\n", msg.ErrCode, msg.ErrCode)
    result += fmt.Sprintf("  Timestamp: %d\n", msg.TimeStamp)
    result += fmt.Sprintf("  Subject: \"%s\" (len: %d)\n", msg.Subject.Value, msg.Subject.Length)
    result += fmt.Sprintf("  Sender ID: \"%s\" (len: %d)\n", msg.SenderID.Value, msg.SenderID.Length)
    result += fmt.Sprintf("  Recipients: %d\n", msg.RecptID.Count)
    
    for i, recpt := range msg.RecptID.Strings {
        result += fmt.Sprintf("    [%d]: \"%s\" (len: %d)\n", i, recpt.Value, recpt.Length)
    }
    
    if len(msg.RawContents) > 0 {
        result += fmt.Sprintf("  Raw Content: %d bytes\n", len(msg.RawContents))
        maxShow := 32
        if len(msg.RawContents) < maxShow {
            maxShow = len(msg.RawContents)
        }
        result += fmt.Sprintf("  Content (hex): %X", msg.RawContents[:maxShow])
        if len(msg.RawContents) > maxShow {
            result += "..."
        }
        result += "\n"
    }
    
    if msg.MsgContent != nil {
        result += fmt.Sprintf("  Parsed Content: %v\n", msg.MsgContent.String())
    }
    
    return result
}