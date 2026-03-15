package outbound

import (
	"encoding/json"
	"math"

	lua "github.com/yuin/gopher-lua"
)

func registerJsonModule(L *lua.LState, musMod *lua.LTable) {
	jsonMod := L.NewTable()

	jsonMod.RawSetString("encode", L.NewFunction(func(L *lua.LState) int {
		lv := L.Get(1)
		goVal := luaValueToJsonable(L, lv)
		data, err := json.Marshal(goVal)
		if err != nil {
			L.RaiseError("json.encode: %s", err.Error())
			return 0
		}
		L.Push(lua.LString(string(data)))
		return 1
	}))

	jsonMod.RawSetString("decode", L.NewFunction(func(L *lua.LState) int {
		str := L.CheckString(1)
		var goVal interface{}
		if err := json.Unmarshal([]byte(str), &goVal); err != nil {
			L.RaiseError("json.decode: %s", err.Error())
			return 0
		}
		L.Push(goValueToLua(L, goVal))
		return 1
	}))

	musMod.RawSetString("json", jsonMod)
}

func isSequentialTable(tbl *lua.LTable) bool {
	maxN := tbl.MaxN()
	if maxN == 0 {
		return false
	}
	count := 0
	tbl.ForEach(func(k, v lua.LValue) { count++ })
	return count == maxN
}

func luaValueToJsonable(L *lua.LState, lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		f := float64(v)
		if f == math.Floor(f) && !math.IsInf(f, 0) && !math.IsNaN(f) &&
			f >= math.MinInt64 && f <= math.MaxInt64 {
			return int64(f)
		}
		return f
	case lua.LString:
		return string(v)
	case *lua.LTable:
		if isSequentialTable(v) {
			arr := make([]interface{}, 0, v.MaxN())
			for i := 1; i <= v.MaxN(); i++ {
				arr = append(arr, luaValueToJsonable(L, v.RawGetInt(i)))
			}
			return arr
		}
		obj := make(map[string]interface{})
		v.ForEach(func(k, val lua.LValue) {
			if ks, ok := k.(lua.LString); ok {
				obj[string(ks)] = luaValueToJsonable(L, val)
			}
		})
		return obj
	default:
		L.RaiseError("json.encode: cannot encode %s", lv.Type().String())
		return nil
	}
}

func goValueToLua(L *lua.LState, val interface{}) lua.LValue {
	switch v := val.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(v)
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case []interface{}:
		tbl := L.NewTable()
		for _, item := range v {
			tbl.Append(goValueToLua(L, item))
		}
		return tbl
	case map[string]interface{}:
		tbl := L.NewTable()
		for key, item := range v {
			tbl.RawSetString(key, goValueToLua(L, item))
		}
		return tbl
	default:
		return lua.LNil
	}
}
