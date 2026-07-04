package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"fsos-server/internal/domain/ports"
)

type MetricsServer struct {
	port         string
	bindAddr     string
	sessionStore ports.SessionStore
	logger       ports.Logger
	// mu guards startedAt/server, which are written in the Start() goroutine and
	// read from Shutdown() and the HTTP handlers on other goroutines.
	mu          sync.Mutex
	startedAt   time.Time
	server      *http.Server
	msgCount    atomic.Int64
	msgErrors   atomic.Int64
	rateLimited atomic.Int64
	bannedConns atomic.Int64
}

func NewMetricsServer(port string, bindAddr string, sessionStore ports.SessionStore, logger ports.Logger) *MetricsServer {
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}
	return &MetricsServer{
		port:         port,
		bindAddr:     bindAddr,
		sessionStore: sessionStore,
		logger:       logger,
	}
}

func (m *MetricsServer) IncrementMessages()   { m.msgCount.Add(1) }
func (m *MetricsServer) IncrementErrors()      { m.msgErrors.Add(1) }
func (m *MetricsServer) IncrementRateLimited() { m.rateLimited.Add(1) }
func (m *MetricsServer) IncrementBannedConns() { m.bannedConns.Add(1) }

func (m *MetricsServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", m.handleHealth)
	mux.HandleFunc("/metrics", m.handleMetrics)

	addr := m.bindAddr + ":" + m.port
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	m.mu.Lock()
	m.startedAt = time.Now()
	m.server = srv
	m.mu.Unlock()

	m.logger.Info("Metrics server listening", map[string]interface{}{
		"address": addr,
	})

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server error: %w", err)
	}
	return nil
}

func (m *MetricsServer) Shutdown() {
	m.mu.Lock()
	srv := m.server
	m.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}
}

func (m *MetricsServer) uptime() time.Duration {
	m.mu.Lock()
	started := m.startedAt
	m.mu.Unlock()
	return time.Since(started)
}

func (m *MetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"uptime": m.uptime().String(),
	})
}

func (m *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var activeConnections int
	conns, err := m.sessionStore.GetAllConnections()
	if err == nil {
		activeConnections = len(conns)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uptime_seconds":     m.uptime().Seconds(),
		"active_connections": activeConnections,
		"messages_processed": m.msgCount.Load(),
		"message_errors":     m.msgErrors.Load(),
		"rate_limited":       m.rateLimited.Load(),
		"banned_connections": m.bannedConns.Load(),
	})
}
