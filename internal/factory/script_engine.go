package factory

import (
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func NewScriptEngine(scriptsPath string, logger ports.Logger) ports.ScriptEngine {
	if scriptsPath == "" {
		return nil
	}
	return outbound.NewLuaScriptEngine(scriptsPath, logger)
}
