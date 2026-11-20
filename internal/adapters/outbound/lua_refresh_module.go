package outbound

import (
	"musgos/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

func registerRefreshModule(L *lua.LState, mus *lua.LTable, store ports.RefreshTokenStore) {
	mod := L.NewTable()
	mod.RawSetString("issue", L.NewFunction(func(L *lua.LState) int {
		token, err := store.Issue(int64(L.CheckNumber(1)), L.OptString(2, ""))
		if err != nil {
			L.RaiseError("refresh issue failed")
			return 0
		}
		L.Push(lua.LString(token))
		return 1
	}))
	mod.RawSetString("rotate", L.NewFunction(func(L *lua.LState) int {
		id, token, err := store.Rotate(L.CheckString(1))
		if err != nil {
			L.RaiseError("refresh rotation failed")
			return 0
		}
		if id == 0 {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lua.LNumber(id))
		L.Push(lua.LString(token))
		return 2
	}))
	mod.RawSetString("revoke", L.NewFunction(func(L *lua.LState) int {
		if err := store.Revoke(L.CheckString(1)); err != nil {
			L.RaiseError("refresh revocation failed")
		}
		return 0
	}))
	mus.RawSetString("refresh", mod)
}
