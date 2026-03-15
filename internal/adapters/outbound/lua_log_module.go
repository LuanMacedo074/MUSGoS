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
		fields := luaTableToFields(L, 2)
		if fields != nil {
			logFn(msg, fields)
		} else {
			logFn(msg)
		}
		return 0
	}
}

func luaTableToFields(L *lua.LState, argPos int) map[string]interface{} {
	arg := L.Get(argPos)
	tbl, ok := arg.(*lua.LTable)
	if !ok {
		return nil
	}

	fields := make(map[string]interface{})
	tbl.ForEach(func(key, value lua.LValue) {
		k, ok := key.(lua.LString)
		if !ok {
			return
		}
		switch v := value.(type) {
		case lua.LBool:
			fields[string(k)] = bool(v)
		case lua.LNumber:
			fields[string(k)] = float64(v)
		case lua.LString:
			fields[string(k)] = string(v)
		default:
			fields[string(k)] = v.String()
		}
	})
	return fields
}
