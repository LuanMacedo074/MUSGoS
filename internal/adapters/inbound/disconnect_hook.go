package inbound

import (
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// NewDisconnectFlushHook returns an OnDisconnect callback that runs the Lua
// `subject` with the disconnecting client's id as the sender. It lets a script
// flush hot-state (position/vitals/gold) to the DB on disconnect instead of
// waiting for the client's periodic save ping.
//
// Returns nil when disabled (no engine or empty subject — the empty subject is
// how DISCONNECT_HOOK="" turns the feature off). The callback itself is a no-op
// for an empty id or when the script is absent — a client that never logged on
// (whose id is a raw IP:port) simply has no such script/state to flush.
func NewDisconnectFlushHook(engine ports.ScriptEngine, subject string, logger ports.Logger) func(clientID string) {
	if engine == nil || subject == "" {
		return nil
	}
	return func(clientID string) {
		if clientID == "" || !engine.HasScript(subject) {
			return
		}
		if _, err := engine.Execute(&ports.ScriptMessage{
			Subject:  subject,
			SenderID: clientID,
			Content:  lingo.NewLVoid(),
		}); err != nil && logger != nil {
			logger.Error("disconnect flush hook failed", map[string]interface{}{
				"clientID": clientID,
				"subject":  subject,
				"error":    err.Error(),
			})
		}
	}
}
