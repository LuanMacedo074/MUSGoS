package testutil

import "fsos-server/internal/domain/types/lingo"

// HasCall reports whether a recorded message matches the recipient and subject.
// An empty content argument matches any content.
func HasCall(sender *MockMessageSender, recipient, subject, content string) bool {
	for _, c := range sender.Calls {
		if c.RecipientID == recipient && c.Subject == subject &&
			(content == "" || lingo.StringValue(c.Content) == content) {
			return true
		}
	}
	return false
}
