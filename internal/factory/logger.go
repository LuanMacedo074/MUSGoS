package factory

import (
	"fmt"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func ParseLogLevel(level string) ports.LogLevel {
	switch level {
	case "DEBUG":
		return ports.DEBUG
	case "INFO":
		return ports.INFO
	case "WARN":
		return ports.WARN
	case "ERROR":
		return ports.ERROR
	default:
		return ports.INFO
	}
}

func NewLogger(loggerType, component string, level ports.LogLevel, logPath string, bufferSize int) (ports.Logger, error) {
	switch loggerType {
	case "file":
		return outbound.NewFileLogger(component, level, logPath, bufferSize), nil
	default:
		return nil, fmt.Errorf("unsupported logger type: %s", loggerType)
	}
}
