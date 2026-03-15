package outbound

import (
	"fsos-server/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

func registerLogModule(L *lua.LState, musMod *lua.LTable, logger ports.Logger) {
	logMod := L.NewTable()

	logMod.RawSetString("debug", L.NewFunction(makeLogFn(logger.Debug)))
	logMod.RawSetString("info", L.NewFunction(makeLogFn(logger.Info)))
	logMod.RawSetString("warn", L.NewFunction(makeLogFn(logger.Warn)))
	logMod.RawSetString("error", L.NewFunction(makeLogFn(logger.Error)))
	logMod.RawSetString("fatal", L.NewFunction(makeLogFn(logger.Fatal)))

	musMod.RawSetString("log", logMod)
}

func makeLogFn(logFn func(string, ...map[string]interface{})) lua.LGFunction {
	return func(L *lua.LState) int {
		msg := L.CheckString(1)
		fields := luaArgsToFields(L, 2)
		if fields != nil {
			logFn(msg, fields)
		} else {
			logFn(msg)
		}
		return 0
	}
}

// luaArgsToFields accepts either a table or variadic key-value pairs:
//
//	mus.log.info("msg", {key="val"})
//	mus.log.info("msg", "key", val, "key2", val2)
func luaArgsToFields(L *lua.LState, argPos int) map[string]interface{} {
	arg := L.Get(argPos)
	if arg == lua.LNil {
		return nil
	}

	// Table form: mus.log.info("msg", {key="val"})
	if tbl, ok := arg.(*lua.LTable); ok {
		fields := make(map[string]interface{})
		tbl.ForEach(func(key, value lua.LValue) {
			k, ok := key.(lua.LString)
			if !ok {
				return
			}
			fields[string(k)] = luaValueToGo(value)
		})
		return fields
	}

	// Variadic form: mus.log.info("msg", "key", val, "key2", val2)
	n := L.GetTop()
	if argPos > n {
		return nil
	}
	fields := make(map[string]interface{})
	for i := argPos; i <= n-1; i += 2 {
		key := L.Get(i)
		val := L.Get(i + 1)
		if k, ok := key.(lua.LString); ok {
			fields[string(k)] = luaValueToGo(val)
		}
	}
	return fields
}

func luaValueToGo(value lua.LValue) interface{} {
	switch v := value.(type) {
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	default:
		return v.String()
	}
}
