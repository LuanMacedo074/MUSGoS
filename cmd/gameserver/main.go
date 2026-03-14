package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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
		"port":          cfg.Port,
		"log_level":     cfg.LogLevel,
		"logger":        cfg.LoggerType,
		"log_path":      cfg.LogPath,
		"cipher":        cfg.CipherType,
		"protocol":      cfg.Protocol,
		"auth_mode":     cfg.AuthMode,
		"env":           cfg.Environment,
		"database":      cfg.DatabaseType,
		"session_store": cfg.SessionStoreType,
		"cache":         cfg.CacheType,
		"queue":         cfg.QueueType,
		"udp_port":      cfg.UDPPort,
		"rate_limit":    cfg.RateLimitRequests,
		"metrics_port":  cfg.MetricsPort,
		"idle_timeout":  cfg.IdleTimeout,
	})

	dbResult, err := factory.NewDatabase(cfg.DatabaseType, cfg.DatabasePath, migrations.All)
	if err != nil {
		gameLogger.Fatal("Failed to initialize database", map[string]interface{}{
			"error": err,
		})
	}
	defer dbResult.Adapter.Close()

	migrationResult, err := dbResult.MigrationRunner.RunPending()
	if err != nil {
		dbResult.Adapter.Close()
		gameLogger.Fatal("Failed to run migrations", map[string]interface{}{
			"error": err,
		})
	}
	gameLogger.Info("Database initialized", map[string]interface{}{
		"type": cfg.DatabaseType,
		"path": cfg.DatabasePath,
	})
	if len(migrationResult.Ran) > 0 {
		gameLogger.Info("Migrations executed", map[string]interface{}{
			"total":    migrationResult.Total,
			"applied":  migrationResult.Applied + len(migrationResult.Ran),
			"executed": migrationResult.Ran,
		})
	} else {
		gameLogger.Info("Migrations up to date", map[string]interface{}{
			"total":   migrationResult.Total,
			"applied": migrationResult.Applied,
		})
	}

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
	// 1. BanChecker — uses DB + Cache
	banChecker := inbound.NewBanChecker(dbResult.Adapter, cache)

	// 2. ConnPool — standalone, no dependencies
	pool := inbound.NewConnPool()

	// 3. Sender — uses pool as ConnectionWriter
	sender := mus.NewSender(pool, sessionStore, gameLogger, cipher, cfg.AllEncrypted)

	// 4. ScriptEngine — can send messages via Sender + access DB + server info + cache
	scriptEngine := factory.NewScriptEngine(cfg.ScriptsPath, gameLogger, cfg.ScriptTimeout, queue, sender, dbResult.Adapter, dbResult.QueryBuilder, sessionStore, cache)
	if cfg.ScriptsPath != "" {
		scripts, _ := filepath.Glob(filepath.Join(cfg.ScriptsPath, "*.lua"))
		gameLogger.Info("Script engine initialized", map[string]interface{}{
			"scripts_path":   cfg.ScriptsPath,
			"scripts_loaded": len(scripts),
		})
	} else {
		gameLogger.Info("Script engine disabled (no scripts path configured)")
	}

	var registeredTopics []string
	for _, q := range queues.All {
		topic := q.Topic
		h := q.Handler
		queue.Subscribe(topic, func(msg ports.QueueMessage) {
			h(msg.Payload)
		})
		registeredTopics = append(registeredTopics, topic)
	}
	if len(registeredTopics) > 0 {
		gameLogger.Info("Queue consumers registered", map[string]interface{}{
			"count":  len(registeredTopics),
			"topics": registeredTopics,
		})
	}

	// 5. Signal channel for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// 6. TimerManager
	timerManager := inbound.NewTimerManager(sessionStore, pool, gameLogger, func() {
		c <- syscall.SIGTERM
	})
	defer timerManager.Stop()

	// 7. Handler — Dispatcher receives ScriptEngine + Sender + pool
	handler, err := factory.NewHandler(cfg.Protocol, gameLogger, cipher, scriptEngine, dbResult.Adapter, sessionStore, queue, pool, sender, cfg.AuthMode, cfg.DefaultUserLevel, cfg.AllEncrypted, cfg.CommandLevels, nil, timerManager)
	if err != nil {
		gameLogger.Fatal("Failed to initialize protocol handler", map[string]interface{}{
			"error": err,
		})
	}
	gameLogger.Info("Protocol handler initialized", map[string]interface{}{
		"type": cfg.Protocol,
	})

	// 8. Rate Limiter (optional)
	var rateLimiter ports.RateLimiter
	if cfg.RateLimitRequests > 0 {
		rl := inbound.NewRateLimiter(inbound.RateLimiterConfig{
			MaxRequests: cfg.RateLimitRequests,
			Window:      time.Duration(cfg.RateLimitWindow) * time.Second,
		})
		defer rl.Stop()
		rateLimiter = rl
		gameLogger.Info("Rate limiter enabled", map[string]interface{}{
			"max_requests": cfg.RateLimitRequests,
			"window_secs":  cfg.RateLimitWindow,
		})
	}

	// 9. Metrics Server (optional)
	var metrics ports.Metrics
	if cfg.MetricsPort != "" {
		ms := inbound.NewMetricsServer(cfg.MetricsPort, cfg.MetricsBindAddr, sessionStore, gameLogger)
		go func() {
			if err := ms.Start(); err != nil {
				gameLogger.Error("Metrics server error", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}()
		defer ms.Shutdown()
		metrics = ms
	}

	// 10. TCPServer — fully constructed
	server := inbound.NewTCPServer(inbound.TCPServerConfig{
		Port:           cfg.Port,
		ServerIP:       cfg.ServerIP,
		MaxMessageSize: cfg.MaxMessageSize,
		TCPNoDelay:     cfg.TCPNoDelay,
	}, inbound.TCPServerDeps{
		Handler:      handler,
		Pool:         pool,
		Logger:       gameLogger,
		SessionStore: sessionStore,
		BanChecker:   banChecker,
		RateLimiter:  rateLimiter,
		Metrics:      metrics,
	})

	serverReady := make(chan struct{})
	go func() {
		if err := server.Start(serverReady); err != nil {
			gameLogger.Fatal("Failed to start server", map[string]interface{}{
				"error": err,
			})
		}
	}()

	<-serverReady

	// 11. Idle Checker
	if cfg.IdleTimeout > 0 {
		idleChecker := inbound.NewIdleChecker(sessionStore, pool, gameLogger, time.Duration(cfg.IdleTimeout)*time.Second)
		idleChecker.Start()
		defer idleChecker.Stop()
		gameLogger.Info("Idle checker enabled", map[string]interface{}{
			"timeout_seconds": cfg.IdleTimeout,
		})
	}

	// 12. UDP Server
	var udpServer *inbound.UDPServer
	if cfg.UDPPort != "" {
		udpServer = inbound.NewUDPServer(inbound.UDPServerConfig{
			Port:           cfg.UDPPort,
			ServerIP:       cfg.ServerIP,
			MaxMessageSize: cfg.MaxMessageSize,
		}, inbound.UDPServerDeps{
			Handler:     handler,
			Logger:      gameLogger,
			BanChecker:  banChecker,
			RateLimiter: rateLimiter,
			Metrics:     metrics,
		})

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
