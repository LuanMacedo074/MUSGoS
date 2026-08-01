package smus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// standard MUS header, present in every message
var MUSHeader = []byte{0x72, 0x00}

type MUSMessage struct {
	ContentSize       int32
	ErrCode           int32
	TimeStamp         int32
	Subject           MUSMsgHeaderString
	SenderID          MUSMsgHeaderString
	RecptID           MUSMsgHeaderStringList
	RawContents       []byte
	DecryptedContents []byte
	MsgContent        lingo.LValue
}

func ParseMUSMessage(rawmsg []byte) (*MUSMessage, error) {
	return ParseMUSMessageWithDecryption(rawmsg, nil)
}

func ParseMUSMessageWithDecryption(rawmsg []byte, decrypt ports.Cipher) (*MUSMessage, error) {
	if len(rawmsg) < 14 {
		return nil, errors.New("message too short")
	}

	// verify the MUS header
	if rawmsg[0] != MUSHeader[0] || rawmsg[1] != MUSHeader[1] {
		// #All mode: entire packet is encrypted, try decrypting
		if decrypt != nil {
			rawmsg = decrypt.Decrypt(rawmsg)
			if len(rawmsg) < 14 || rawmsg[0] != MUSHeader[0] || rawmsg[1] != MUSHeader[1] {
				return nil, fmt.Errorf("invalid MUS header after decryption: expected %X, got %X", MUSHeader, rawmsg[:2])
			}
		} else {
			return nil, fmt.Errorf("invalid MUS header: expected %X, got %X", MUSHeader, rawmsg[:2])
		}
	}

	// skip the first 6 bytes (standard header + message size); parsing begins at offset 6
	msg := &MUSMessage{}
	readPtr := 2

	// read the content size
	msg.ContentSize = int32(binary.BigEndian.Uint32(rawmsg[readPtr:]))
	readPtr += 4

	// A ContentSize >= 0x80000000 turns negative as int32; guard that (and compute
	// the total as int, not int32, so 6+ContentSize can't overflow) before using it
	// as a length (L4).
	if msg.ContentSize < 0 || len(rawmsg) < 6+int(msg.ContentSize) {
		return nil, fmt.Errorf("message truncated or invalid content size: %d (have %d bytes)", msg.ContentSize, len(rawmsg))
	}

	// required header fields
	msg.ErrCode = int32(binary.BigEndian.Uint32(rawmsg[readPtr:]))
	readPtr += 4

	msg.TimeStamp = int32(binary.BigEndian.Uint32(rawmsg[readPtr:]))
	readPtr += 4

	// header strings use their own extraction methods
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

	if readPtr < len(rawmsg) {
		remainingBytes := rawmsg[readPtr:]
		content := remainingBytes

		if msg.Subject.Value == "Logon" && decrypt != nil {
			content = decrypt.Decrypt(remainingBytes)
		}

		msg.RawContents = remainingBytes
		msg.DecryptedContents = content
		msg.MsgContent = lingo.FromRawBytes(content, 0)
	}

	return msg, nil
}

func (msg *MUSMessage) GetBytes() []byte {
	var payload bytes.Buffer

	binary.Write(&payload, binary.BigEndian, msg.ErrCode)
	binary.Write(&payload, binary.BigEndian, msg.TimeStamp)
	msg.Subject.WriteBytes(&payload)
	msg.SenderID.WriteBytes(&payload)
	msg.RecptID.WriteBytes(&payload)

	if msg.MsgContent != nil {
		payload.Write(msg.MsgContent.GetBytes())
	}

	var buf bytes.Buffer
	buf.Write(MUSHeader)
	binary.Write(&buf, binary.BigEndian, int32(payload.Len()))
	buf.Write(payload.Bytes())

	return buf.Bytes()
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

	return result
}
