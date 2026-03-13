package factory

import (
	"fmt"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
)

func NewHandler(protocol string, log ports.Logger, cipher ports.Cipher, scriptEngine ports.ScriptEngine, db ports.DBAdapter, sessionStore ports.SessionStore, authMode string, defaultUserLevel int) (ports.MessageHandler, error) {
	switch protocol {
	case "smus":
		logonService := mus.NewLogonService(db, sessionStore, cipher, log, authMode, defaultUserLevel)
		return inbound.NewSMUSHandler(log, cipher, scriptEngine, logonService), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
