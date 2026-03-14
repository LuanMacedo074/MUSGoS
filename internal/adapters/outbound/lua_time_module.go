package outbound

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

func registerTimeModule(L *lua.LState, musMod *lua.LTable) {
	timeMod := L.NewTable()

	// mus.time.now() -> number (Unix timestamp in seconds)
	timeMod.RawSetString("now", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(time.Now().Unix()))
		return 1
	}))

	musMod.RawSetString("time", timeMod)
}
