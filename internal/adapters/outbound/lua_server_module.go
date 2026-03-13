package outbound

import (
	"fsos-server/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

func registerServerModule(L *lua.LState, musMod *lua.LTable, sessionStore ports.SessionStore) {
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

	musMod.RawSetString("server", serverMod)
}
