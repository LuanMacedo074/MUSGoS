package outbound

import (
	"time"

	"fsos-server/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

func registerCacheModule(L *lua.LState, musMod *lua.LTable, cache ports.Cache) {
	cacheMod := L.NewTable()

	// mus.cache.get(key) -> string|nil
	cacheMod.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		data, err := cache.Get(key)
		if err != nil || data == nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lua.LString(string(data)))
		return 1
	}))

	// mus.cache.set(key, value, [ttlSeconds])
	cacheMod.RawSetString("set", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		value := L.CheckString(2)
		ttlSeconds := L.OptInt(3, 0)
		ttl := time.Duration(ttlSeconds) * time.Second
		if err := cache.Set(key, []byte(value), ttl); err != nil {
			L.Push(lua.LBool(false))
			return 1
		}
		L.Push(lua.LBool(true))
		return 1
	}))

	// mus.cache.delete(key)
	cacheMod.RawSetString("delete", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		if err := cache.Delete(key); err != nil {
			L.Push(lua.LBool(false))
			return 1
		}
		L.Push(lua.LBool(true))
		return 1
	}))

	// mus.cache.exists(key) -> bool
	cacheMod.RawSetString("exists", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		exists, err := cache.Exists(key)
		if err != nil {
			L.Push(lua.LBool(false))
			return 1
		}
		L.Push(lua.LBool(exists))
		return 1
	}))

	musMod.RawSetString("cache", cacheMod)
}
