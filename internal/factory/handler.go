package factory

import (
	"fmt"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
)

func NewHandler(
	protocol string,
	log ports.Logger,
	cipher ports.Cipher,
	scriptEngine ports.ScriptEngine,
	db ports.DBAdapter,
	sessionStore ports.SessionStore,
	queue ports.QueuePublisher,
	connWriter ports.ConnectionWriter,
	sender *mus.Sender,
	authMode string,
	defaultUserLevel int,
	allEncrypted bool,
) (ports.MessageHandler, error) {
	switch protocol {
	case "smus":
		movieManager := mus.NewMovieManager(sessionStore, log)
		groupManager := mus.NewGroupManager(sessionStore, log)
		systemService := mus.NewSystemService(db, sessionStore, cipher, log, movieManager, groupManager, connWriter, authMode, defaultUserLevel)
		dispatcher := mus.NewDispatcher(log, scriptEngine, systemService, sender, queue)
		return inbound.NewSMUSHandler(log, cipher, dispatcher, allEncrypted), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
