package ports

import "fsos-server/internal/domain/types/lingo"

type MessageSender interface {
	SendMessage(senderID, recipientID, subject string, content lingo.LValue) error
	// SendMessageFrom delivers like SendMessage but splits the two roles the
	// senderID plays: `wireFrom` is stamped as the protocol sender the client
	// sees, while `routingSender` resolves delivery context (the sender's
	// movie for @group fan-out). Script-authored messages use this to appear
	// as system.script — the unchanged client only renders several subjects
	// when they come from the server — while still routing by the invoking
	// player's session.
	SendMessageFrom(wireFrom, routingSender, recipientID, subject string, content lingo.LValue) error
}
