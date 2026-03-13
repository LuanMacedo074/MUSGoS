package factory

import (
	"fmt"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/domain/ports"
)

func NewHandler(protocol string, log ports.Logger, cipher ports.Cipher, scriptEngine ports.ScriptEngine) (ports.MessageHandler, error) {
	switch protocol {
	case "smus":
		return inbound.NewSMUSHandler(log, cipher, scriptEngine), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
