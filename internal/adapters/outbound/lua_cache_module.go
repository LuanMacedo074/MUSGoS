package outbound

import (
	"time"

	"fsos-server/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

func registerCacheModule(L *lua.LState, musMod *lua.LTable, cache ports.Cache) func() error {
	cacheMod := L.NewTable()
	locks := make(map[string]*ports.CacheLock)
	scopedDepth := 0
	// Locks belong to this execution, including early returns and script errors.
	cacheMod.RawSetString("lock", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		if locks[key] != nil {
			return 0
		}
		if scopedDepth > 0 {
			L.RaiseError("cannot acquire another lock inside a scoped lock")
			return 0
		}
		lease, err := cache.AcquireLock(L.Context(), key)
		if err != nil {
			L.RaiseError("cache lock: %v", err)
			return 0
		}
		locks[key] = lease
		L.SetContext(lease.Context)
		return 0
	}))

	// A scoped lease ends before the script continues, including callback errors.
	cacheMod.RawSetString("withLock", L.NewFunction(func(L *lua.LState) int {
		key, callback := L.CheckString(1), L.CheckFunction(2)
		previousContext := L.Context()
		lease := locks[key]
		owned := lease != nil
		if !owned {
			if scopedDepth > 0 {
				L.RaiseError("cannot acquire another lock inside a scoped lock")
				return 0
			}
			var err error
			lease, err = cache.AcquireLock(previousContext, key)
			if err != nil {
				L.RaiseError("cache lock: %v", err)
				return 0
			}
			locks[key] = lease
			L.SetContext(lease.Context)
		}
		top := L.GetTop()
		scopedDepth++
		err := L.CallByParam(lua.P{Fn: callback, NRet: lua.MultRet, Protect: true})
		scopedDepth--
		if !owned {
			releaseErr := lease.Release()
			delete(locks, key)
			L.SetContext(previousContext)
			if err == nil {
				err = releaseErr
			}
		}
		if err != nil {
			L.RaiseError("cache lock callback: %v", err)
			return 0
		}
		return L.GetTop() - top
	}))

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

	// mus.cache.setAdd(key, member) -> bool
	cacheMod.RawSetString("setAdd", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		member := L.CheckString(2)
		if err := cache.SetAdd(key, member); err != nil {
			L.Push(lua.LBool(false))
			return 1
		}
		L.Push(lua.LBool(true))
		return 1
	}))

	// mus.cache.setRemove(key, member) -> bool
	cacheMod.RawSetString("setRemove", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		member := L.CheckString(2)
		if err := cache.SetRemove(key, member); err != nil {
			L.Push(lua.LBool(false))
			return 1
		}
		L.Push(lua.LBool(true))
		return 1
	}))

	// mus.cache.setMembers(key) -> table|nil
	cacheMod.RawSetString("setMembers", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		members, err := cache.SetMembers(key)
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		tbl := L.NewTable()
		for _, m := range members {
			tbl.Append(lua.LString(m))
		}
		L.Push(tbl)
		return 1
	}))

	// mus.cache.setIsMember(key, member) -> bool
	cacheMod.RawSetString("setIsMember", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		member := L.CheckString(2)
		isMember, err := cache.SetIsMember(key, member)
		if err != nil {
			L.Push(lua.LBool(false))
			return 1
		}
		L.Push(lua.LBool(isMember))
		return 1
	}))

	musMod.RawSetString("cache", cacheMod)
	return func() error {
		var firstErr error
		for _, lease := range locks {
			if err := lease.Release(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}
}
