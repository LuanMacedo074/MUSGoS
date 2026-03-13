package lingo

import (
	"math"
	"sort"

	lua "github.com/yuin/gopher-lua"
)

// LValueToLua converts a lingo LValue to a gopher-lua value.
func LValueToLua(L *lua.LState, val LValue) lua.LValue {
	if val == nil {
		return lua.LNil
	}

	switch v := val.(type) {
	case *LInteger:
		return lua.LNumber(v.Value)
	case *LFloat:
		return lua.LNumber(v.Value)
	case *LString:
		return lua.LString(v.Value)
	case *LSymbol:
		return lua.LString(v.Value)
	case *LList:
		tbl := L.NewTable()
		for _, elem := range v.Values {
			tbl.Append(LValueToLua(L, elem))
		}
		return tbl
	case *LPropList:
		tbl := L.NewTable()
		for i := 0; i < len(v.Properties); i++ {
			key := v.Properties[i].String()
			tbl.RawSetString(key, LValueToLua(L, v.Values[i]))
		}
		return tbl
	case *LVoid:
		return lua.LNil
	default:
		return lua.LNil
	}
}

// LuaToLValue converts a gopher-lua value to a lingo LValue.
func LuaToLValue(lv lua.LValue) LValue {
	switch v := lv.(type) {
	case lua.LBool:
		if v {
			return NewLInteger(1)
		}
		return NewLInteger(0)
	case lua.LNumber:
		f := float64(v)
		if f == math.Trunc(f) && f >= math.MinInt32 && f <= math.MaxInt32 {
			return NewLInteger(int32(f))
		}
		return NewLFloat(f)
	case lua.LString:
		return NewLString(string(v))
	case *lua.LTable:
		return luaTableToLValue(v)
	case *lua.LNilType:
		return NewLVoid()
	default:
		return NewLVoid()
	}
}

// luaTableToLValue converts a Lua table to either LList or LPropList.
// If the table has sequential integer keys 1..N, it becomes an LList.
// Otherwise, it becomes an LPropList with LSymbol keys.
func luaTableToLValue(tbl *lua.LTable) LValue {
	maxN := tbl.MaxN()

	if maxN > 0 && isSequentialArray(tbl, maxN) {
		list := NewLList()
		for i := 1; i <= maxN; i++ {
			list.Values = append(list.Values, LuaToLValue(tbl.RawGetInt(i)))
		}
		return list
	}

	// Collect all string keys
	props := NewLPropList()
	var keys []string
	tbl.ForEach(func(k, v lua.LValue) {
		if str, ok := k.(lua.LString); ok {
			keys = append(keys, string(str))
		}
	})
	sort.Strings(keys)

	for _, key := range keys {
		val := tbl.RawGetString(key)
		props.Properties = append(props.Properties, NewLSymbol(key))
		props.Values = append(props.Values, LuaToLValue(val))
	}

	return props
}

// isSequentialArray checks if a table has only sequential integer keys 1..maxN.
func isSequentialArray(tbl *lua.LTable, maxN int) bool {
	count := 0
	tbl.ForEach(func(k, v lua.LValue) {
		count++
	})
	return count == maxN
}
