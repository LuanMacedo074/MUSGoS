package outbound

import (
	"musgos/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

// registerTokenModule exposes single-use tokens with a user_id and string claims.
func registerTokenModule(L *lua.LState, musMod *lua.LTable, store ports.TokenStore) {
	mod := L.NewTable()
	mod.RawSetString("user_level_claim", lua.LString(ports.TokenUserLevelClaim))
	mod.RawSetString("issue", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		payload := ports.TokenPayload{Claims: map[string]string{}}
		if id, ok := tbl.RawGetString("user_id").(lua.LString); ok {
			payload.UserID = string(id)
		}
		if claims, ok := tbl.RawGetString("claims").(*lua.LTable); ok {
			claims.ForEach(func(k, v lua.LValue) {
				key, keyOK := k.(lua.LString)
				value, valueOK := v.(lua.LString)
				if !keyOK || !valueOK {
					L.RaiseError("token claims must be strings")
					return
				}
				payload.Claims[string(key)] = string(value)
			})
		}
		token, err := store.Issue(payload)
		if err != nil {
			L.RaiseError("token.issue failed: %s", err)
			return 0
		}
		L.Push(lua.LString(token))
		return 1
	}))
	mod.RawSetString("validate", L.NewFunction(func(L *lua.LState) int {
		payload, err := store.Validate(L.CheckString(1))
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		res, claims := L.NewTable(), L.NewTable()
		res.RawSetString("user_id", lua.LString(payload.UserID))
		for k, v := range payload.Claims {
			claims.RawSetString(k, lua.LString(v))
		}
		res.RawSetString("claims", claims)
		L.Push(res)
		return 1
	}))
	musMod.RawSetString("token", mod)
}
