package factory

import (
	"fmt"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/domain/ports"
)

func NewHandler(protocol string, log ports.Logger, cipher ports.Cipher) (ports.MessageHandler, error) {
	switch protocol {
	case "smus":
		return inbound.NewSMUSHandler(log, cipher), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
