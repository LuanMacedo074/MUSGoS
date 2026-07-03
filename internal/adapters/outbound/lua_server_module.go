package outbound

import (
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	lua "github.com/yuin/gopher-lua"
)

// systemScriptSender is the protocol "from" for server-authored broadcasts.
// The unchanged client only renders subjects like "Broadcast" when they arrive
// from system.script (the invoking player's name goes in the content instead).
const systemScriptSender = "system.script"

// registerServerModule builds mus.server. sender may be nil in contexts without
// an outbound path (e.g. tests) — broadcast degrades to a no-op.
func registerServerModule(L *lua.LState, musMod *lua.LTable, sessionStore ports.SessionStore, sender ports.MessageSender, logger ports.Logger) {
	serverMod := L.NewTable()

	serverMod.RawSetString("getUserCount", L.NewFunction(func(L *lua.LState) int {
		conns, err := sessionStore.GetAllConnections()
		if err != nil {
			L.Push(lua.LNumber(0))
			return 1
		}
		L.Push(lua.LNumber(len(conns)))
		return 1
	}))

	// mus.server.isOnline(name) -> bool. `name` is the post-Logon userID, i.e.
	// the same string a script sees via mus.getSender() and the session key.
	serverMod.RawSetString("isOnline", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		online, err := sessionStore.IsConnected(name)
		if err != nil {
			L.Push(lua.LFalse)
			return 1
		}
		L.Push(lua.LBool(online))
		return 1
	}))

	// mus.server.broadcast(subject, content) sends to every online user. There
	// is no global room, so we fan out over all connections. Per-recipient
	// failures are logged and skipped (like the group fan-out); no-op if there
	// is no outbound sender.
	serverMod.RawSetString("broadcast", L.NewFunction(func(L *lua.LState) int {
		subject := L.CheckString(1)
		lingoContent := lingo.LuaToLValue(L.Get(2))
		if sender == nil {
			return 0
		}
		conns, err := sessionStore.GetAllConnections()
		if err != nil {
			if logger != nil {
				logger.Error("mus.server.broadcast: failed to list connections", map[string]interface{}{
					"error": err.Error(),
				})
			}
			return 0
		}
		for _, c := range conns {
			if err := sender.SendMessage(systemScriptSender, c.ClientID, subject, lingoContent); err != nil && logger != nil {
				logger.Warn("mus.server.broadcast: delivery failed", map[string]interface{}{
					"recipient": c.ClientID,
					"subject":   subject,
					"error":     err.Error(),
				})
			}
		}
		return 0
	}))

	musMod.RawSetString("server", serverMod)
}
