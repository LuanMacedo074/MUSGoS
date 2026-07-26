package outbound

import (
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/crypto/bcrypt"
)

// registerBcryptModule exposes mus.bcrypt — the password-verification primitive
// the HTTP login script needs to compare a submitted password against a stored
// bcrypt hash. Hashing on write already lives in mus.db.createUser/setPassword.
func registerBcryptModule(L *lua.LState, musMod *lua.LTable) {
	mod := L.NewTable()

	// mus.bcrypt.verify(plaintext, hash) -> bool
	mod.RawSetString("verify", L.NewFunction(func(L *lua.LState) int {
		plaintext := L.CheckString(1)
		hash := L.CheckString(2)
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
		L.Push(lua.LBool(err == nil))
		return 1
	}))

	musMod.RawSetString("bcrypt", mod)
}
