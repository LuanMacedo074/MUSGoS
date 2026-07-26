package factory

import (
	"fmt"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
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
	commandLevels map[string]int,
	emailSender ports.EmailSender,
	timerManager ports.TimerManager,
	tokenStore ports.TokenStore,
) (ports.MessageHandler, error) {
	switch protocol {
	case "smus":
		movieManager := mus.NewMovieManager(sessionStore, log)
		groupManager := mus.NewGroupManager(sessionStore, log)
		var logonOpts []services.LogonOption
		if tokenStore != nil {
			logonOpts = append(logonOpts, services.WithTokenStore(tokenStore))
		}
		logonService := services.NewLogonService(db, sessionStore, connWriter, log, authMode, defaultUserLevel, logonOpts...)
		authorizer := services.NewAuthorizer(sessionStore, commandLevels)
		systemService := mus.NewSystemService(db, sessionStore, log, movieManager, groupManager, connWriter, logonService, authorizer, emailSender, timerManager)
		dispatcher := mus.NewDispatcher(log, scriptEngine, systemService, sender, queue)
		return inbound.NewSMUSHandler(log, cipher, dispatcher, allEncrypted), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
