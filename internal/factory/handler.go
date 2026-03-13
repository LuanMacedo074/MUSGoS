package factory

import (
	"fmt"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
)

func NewHandler(protocol string, log ports.Logger, cipher ports.Cipher, scriptEngine ports.ScriptEngine, db ports.DBAdapter, sessionStore ports.SessionStore, queue ports.QueuePublisher, authMode string, defaultUserLevel int) (ports.MessageHandler, error) {
	switch protocol {
	case "smus":
		movieManager := mus.NewMovieManager(sessionStore, log)
		groupManager := mus.NewGroupManager(sessionStore, log)
		logonService := mus.NewLogonService(db, sessionStore, cipher, log, movieManager, authMode, defaultUserLevel)
		return inbound.NewSMUSHandler(log, cipher, scriptEngine, logonService, movieManager, groupManager, queue), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
