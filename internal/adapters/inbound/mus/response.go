package mus

import (
	"time"

	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func NewResponse(subject string, senderID string, recipients []string, errCode int32, content lingo.LValue) *smus.MUSMessage {
	msg := &smus.MUSMessage{
		ErrCode:   errCode,
		TimeStamp: int32(time.Now().Unix()),
		Subject: smus.MUSMsgHeaderString{
			Length: len(subject),
			Value:  subject,
		},
		SenderID: smus.MUSMsgHeaderString{
			Length: len(senderID),
			Value:  senderID,
		},
		MsgContent: content,
	}

	msg.RecptID.Count = len(recipients)
	msg.RecptID.Strings = make([]smus.MUSMsgHeaderString, len(recipients))
	for i, r := range recipients {
		msg.RecptID.Strings[i] = smus.MUSMsgHeaderString{
			Length: len(r),
			Value:  r,
		}
	}

	return msg
}
