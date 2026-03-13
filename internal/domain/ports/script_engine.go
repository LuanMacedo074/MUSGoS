package ports

import "fsos-server/internal/domain/types/lingo"

// ScriptMessage holds the fields a script needs from a parsed message.
// Protocol-agnostic — the handler extracts these from whatever protocol it handles.
type ScriptMessage struct {
	Subject  string
	SenderID string
	Content  lingo.LValue
}

type ScriptResult struct {
	Content lingo.LValue
}

type ScriptEngine interface {
	HasScript(subject string) bool
	Execute(msg *ScriptMessage) (*ScriptResult, error)
}
