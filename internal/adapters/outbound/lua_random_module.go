package outbound

import lua "github.com/yuin/gopher-lua"

func registerRandomModule(L *lua.LState, musMod *lua.LTable, random func() float64) {
	// mus.random() -> number in [0, 1)
	musMod.RawSetString("random", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(random()))
		return 1
	}))
}
