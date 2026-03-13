package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"fsos-server/external/migrations"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/config"
	"fsos-server/internal/factory"
)

func main() {
	cfg := config.LoadServerConfig()

	gameLogger, err := factory.NewLogger(cfg.LoggerType, cfg.ApplicationName, factory.ParseLogLevel(cfg.LogLevel), cfg.LogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	gameLogger.Info("Starting " + cfg.ApplicationName + "...")
	gameLogger.Info("Server configuration", map[string]interface{}{
		"port":      cfg.Port,
		"log_level": cfg.LogLevel,
		"logger":    cfg.LoggerType,
		"log_path":  cfg.LogPath,
		"cipher":    cfg.CipherType,
		"protocol":  cfg.Protocol,
		"env":       cfg.Environment,
	})

	dbResult, err := factory.NewDatabase(cfg.DatabaseType, cfg.DatabasePath, migrations.All)
	if err != nil {
		gameLogger.Fatal("Failed to initialize database", map[string]interface{}{
			"error": err,
		})
	}
	defer dbResult.Adapter.Close()

	if err := dbResult.MigrationRunner.RunPending(); err != nil {
		dbResult.Adapter.Close()
		gameLogger.Fatal("Failed to run migrations", map[string]interface{}{
			"error": err,
		})
	}
	gameLogger.Info("Database initialized", map[string]interface{}{
		"type": cfg.DatabaseType,
		"path": cfg.DatabasePath,
	})

	cipher, err := factory.NewCipher(cfg.CipherType, cfg.EncryptionKey)
	if err != nil {
		gameLogger.Fatal("Failed to initialize cipher", map[string]interface{}{
			"error": err,
		})
	}
	gameLogger.Info("Cipher initialized", map[string]interface{}{
		"type": cfg.CipherType,
	})

	handler, err := factory.NewHandler(cfg.Protocol, gameLogger, cipher)
	if err != nil {
		gameLogger.Fatal("Failed to initialize protocol handler", map[string]interface{}{
			"error": err,
		})
	}
	gameLogger.Info("Protocol handler initialized", map[string]interface{}{
		"type": cfg.Protocol,
	})

	sessionStore, err := factory.NewSessionStore(cfg.SessionStoreType, cfg.Redis)
	if err != nil {
		gameLogger.Fatal("Failed to initialize session store", map[string]interface{}{
			"error": err,
		})
	}
	defer sessionStore.Close()
	gameLogger.Info("Session store initialized", map[string]interface{}{
		"type": cfg.SessionStoreType,
	})

	server := inbound.NewTCPServer(cfg.Port, gameLogger, handler, sessionStore)

	serverReady := make(chan struct{})
	go func() {
		if err := server.Start(serverReady); err != nil {
			gameLogger.Fatal("Failed to start server", map[string]interface{}{
				"error": err,
			})
		}
	}()

	<-serverReady

	console := inbound.NewConsole(dbResult.Adapter, gameLogger, os.Stdin)
	go console.Run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	gameLogger.Info("Shutting down server...")
	server.Shutdown()
}
