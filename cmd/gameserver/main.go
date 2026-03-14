package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fsos-server/external/migrations"
	"fsos-server/external/queues"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/config"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/factory"
)

func main() {
	cfg := config.LoadServerConfig()

	gameLogger, err := factory.NewLogger(cfg.LoggerType, cfg.ApplicationName, factory.ParseLogLevel(cfg.LogLevel), cfg.LogPath, cfg.LogBufferSize)
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
		"auth_mode": cfg.AuthMode,
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

	queue, err := factory.NewMessageQueue(cfg.QueueType, cfg.QueueRedis, cfg.RabbitMQ)
	if err != nil {
		gameLogger.Fatal("Failed to initialize message queue", map[string]interface{}{
			"error": err,
		})
	}
	defer queue.Close()
	gameLogger.Info("Message queue initialized", map[string]interface{}{
		"type": cfg.QueueType,
	})

	cache, err := factory.NewCache(cfg.CacheType, cfg.CacheRedis)
	if err != nil {
		gameLogger.Fatal("Failed to initialize cache", map[string]interface{}{
			"error": err,
		})
	}
	defer cache.Close()
	gameLogger.Info("Cache initialized", map[string]interface{}{
		"type": cfg.CacheType,
	})
	// 1. ConnPool — standalone, no dependencies
	pool := inbound.NewConnPool()

	// 2. Sender — uses pool as ConnectionWriter
	sender := mus.NewSender(pool, sessionStore, gameLogger, cipher, cfg.AllEncrypted)

	// 3. ScriptEngine — can send messages via Sender + access DB + server info + cache
	scriptEngine := factory.NewScriptEngine(cfg.ScriptsPath, gameLogger, cfg.ScriptTimeout, queue, sender, dbResult.Adapter, dbResult.QueryBuilder, sessionStore, cache)
	gameLogger.Info("Script engine initialized", map[string]interface{}{
		"scripts_path": cfg.ScriptsPath,
	})

	for _, q := range queues.All {
		topic := q.Topic
		h := q.Handler
		queue.Subscribe(topic, func(msg ports.QueueMessage) {
			h(msg.Payload)
		})
		gameLogger.Debug("Registered queue consumer", map[string]interface{}{
			"topic": topic,
		})
	}

	// 4. Signal channel for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// 5. TimerManager
	timerManager := inbound.NewTimerManager(sessionStore, pool, gameLogger, func() {
		c <- syscall.SIGTERM
	})
	defer timerManager.Stop()

	// 6. Handler — Dispatcher receives ScriptEngine + Sender + pool
	handler, err := factory.NewHandler(cfg.Protocol, gameLogger, cipher, scriptEngine, dbResult.Adapter, sessionStore, queue, pool, sender, cfg.AuthMode, cfg.DefaultUserLevel, cfg.AllEncrypted, cfg.CommandLevels, nil, timerManager)
	if err != nil {
		gameLogger.Fatal("Failed to initialize protocol handler", map[string]interface{}{
			"error": err,
		})
	}
	gameLogger.Info("Protocol handler initialized", map[string]interface{}{
		"type": cfg.Protocol,
	})

	// 7. TCPServer — fully constructed, no SetHandler
	server := inbound.NewTCPServer(inbound.TCPServerConfig{
		Port:           cfg.Port,
		ServerIP:       cfg.ServerIP,
		MaxMessageSize: cfg.MaxMessageSize,
		TCPNoDelay:     cfg.TCPNoDelay,
	}, handler, pool, gameLogger, sessionStore)

	serverReady := make(chan struct{})
	go func() {
		if err := server.Start(serverReady); err != nil {
			gameLogger.Fatal("Failed to start server", map[string]interface{}{
				"error": err,
			})
		}
	}()

	<-serverReady

	// 8. Idle Checker
	if cfg.IdleTimeout > 0 {
		idleChecker := inbound.NewIdleChecker(sessionStore, pool, gameLogger, time.Duration(cfg.IdleTimeout)*time.Second)
		idleChecker.Start()
		defer idleChecker.Stop()
		gameLogger.Info("Idle checker enabled", map[string]interface{}{
			"timeout_seconds": cfg.IdleTimeout,
		})
	}

	// 9. UDP Server
	var udpServer *inbound.UDPServer
	if cfg.UDPPort != "" {
		udpServer = inbound.NewUDPServer(inbound.UDPServerConfig{
			Port:           cfg.UDPPort,
			ServerIP:       cfg.ServerIP,
			MaxMessageSize: cfg.MaxMessageSize,
		}, handler, gameLogger)

		udpReady := make(chan struct{})
		go func() {
			if err := udpServer.Start(udpReady); err != nil {
				gameLogger.Fatal("Failed to start UDP server", map[string]interface{}{
					"error": err,
				})
			}
		}()
		<-udpReady
	}

	console := inbound.NewConsole(dbResult.Adapter, gameLogger, os.Stdin, cfg.DefaultUserLevel)
	go console.Run()

	<-c
	gameLogger.Info("Shutting down server...")
	if udpServer != nil {
		udpServer.Shutdown()
	}
	server.Shutdown()
}
