package ports

import "fsos-server/internal/domain/types/lingo"

type MessageSender interface {
	SendMessage(senderID, recipientID, subject string, content lingo.LValue) error
}
