package outbound

import (
	"fsos-server/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

// registerTokenModule exposes mus.token — issue/validate single-use auth tokens
// carrying a {character_name, user_level} payload. The HTTP login script mints a
// token here; the MUS Logon token path validates (consumes) it against the same
// store, so the identity crosses the HTTP→MUS socket gap.
func registerTokenModule(L *lua.LState, musMod *lua.LTable, store ports.TokenStore) {
	mod := L.NewTable()

	// mus.token.issue({ character_name = ..., user_level = ... }) -> token string
	mod.RawSetString("issue", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		payload := ports.TokenPayload{}
		if s, ok := tbl.RawGetString("character_name").(lua.LString); ok {
			payload.CharacterName = string(s)
		}
		if n, ok := tbl.RawGetString("user_level").(lua.LNumber); ok {
			payload.UserLevel = int(n)
		}
		token, err := store.Issue(payload)
		if err != nil {
			L.RaiseError("token.issue failed: %s", err.Error())
			return 0
		}
		L.Push(lua.LString(token))
		return 1
	}))

	// mus.token.validate(token) -> { character_name = ..., user_level = ... } | nil
	mod.RawSetString("validate", L.NewFunction(func(L *lua.LState) int {
		token := L.CheckString(1)
		payload, err := store.Validate(token)
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		res := L.NewTable()
		res.RawSetString("character_name", lua.LString(payload.CharacterName))
		res.RawSetString("user_level", lua.LNumber(payload.UserLevel))
		L.Push(res)
		return 1
	}))

	musMod.RawSetString("token", mod)
}
