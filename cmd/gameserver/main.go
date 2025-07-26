package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"fsos-server/internal/crypto"
	"fsos-server/internal/handlers"
	"fsos-server/internal/protocol"
	"fsos-server/internal/utilities/logger"
)

func main() {
	var port = flag.String("port", getEnv("PORT", "1199"), "TCP port to listen on")
	var logLevel = flag.String("log-level", getEnv("LOG_LEVEL", "INFO"), "Log level (DEBUG, INFO, WARN, ERROR)")
	var encryptionKey = flag.String("encryption_key", getEnv("ENCRYPTION_KEY", "NO_ENCRYPTION_KEY"), "Key to SMUS Encryption")
	flag.Parse()

	gameLogger := logger.New("FSOS-SERVER", parseLogLevel(*logLevel))
	gameLogger.Info("Starting FSOS Game Server...")

	gameLogger.Info("Server configuration", map[string]interface{}{
		"port":          *port,
		"log_level":     *logLevel,
		"env":           getEnv("ENVIRONMENT", "development"),
		"encryptionKey": *encryptionKey,
	})

	blowfish := crypto.NewBlowfish(*encryptionKey)
	blowfish.SetKey()
	smusHandler := handlers.NewSMUSHandler(gameLogger, blowfish)
	server := protocol.NewTCPServer(*port, gameLogger, smusHandler)

	gameLogger.Info("Blowfish parser initialized", map[string]interface{}{
		"key": blowfish.StrKey,
	})

	go func() {
		if err := server.Start(); err != nil {
			gameLogger.Fatal("Failed to start server", map[string]interface{}{
				"error": err,
			})
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	gameLogger.Info("Shutting down server...")
	server.Shutdown()
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "DEBUG":
		return logger.LogLevel(logger.DEBUG)
	case "INFO":
		return logger.LogLevel(logger.INFO)
	case "WARN":
		return logger.LogLevel(logger.WARN)
	case "ERROR":
		return logger.LogLevel(logger.ERROR)
	default:
		return logger.LogLevel(logger.INFO)
	}
}
