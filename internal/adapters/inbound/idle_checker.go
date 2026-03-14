package inbound

import (
	"sync"
	"time"

	"fsos-server/internal/domain/ports"
)

type IdleChecker struct {
	sessionStore ports.SessionStore
	connWriter   ports.ConnectionWriter
	logger       ports.Logger
	timeout      time.Duration
	ticker       *time.Ticker
	done         chan struct{}
	stopOnce     sync.Once
}

func NewIdleChecker(sessionStore ports.SessionStore, connWriter ports.ConnectionWriter, logger ports.Logger, timeout time.Duration) *IdleChecker {
	return &IdleChecker{
		sessionStore: sessionStore,
		connWriter:   connWriter,
		logger:       logger,
		timeout:      timeout,
		done:         make(chan struct{}),
	}
}

func (ic *IdleChecker) Start() {
	interval := ic.timeout / 2
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}

	ic.ticker = time.NewTicker(interval)
	ic.logger.Info("Idle checker started", map[string]interface{}{
		"timeout":  ic.timeout.String(),
		"interval": interval.String(),
	})

	go func() {
		for {
			select {
			case <-ic.done:
				return
			case <-ic.ticker.C:
				ic.check()
			}
		}
	}()
}

func (ic *IdleChecker) check() {
	conns, err := ic.sessionStore.GetAllConnections()
	if err != nil {
		ic.logger.Error("Idle check: failed to get connections", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	now := time.Now()
	for _, conn := range conns {
		lastActivity := conn.LastActivityAt
		if lastActivity.IsZero() {
			lastActivity = conn.ConnectedAt
		}
		if now.Sub(lastActivity) > ic.timeout {
			ic.logger.Info("Disconnecting idle user", map[string]interface{}{
				"clientID":       conn.ClientID,
				"lastActivityAt": lastActivity.String(),
				"idle":           now.Sub(lastActivity).String(),
			})
			ic.sessionStore.LeaveAllRooms(conn.ClientID)
			ic.sessionStore.UnregisterConnection(conn.ClientID)
			if ic.connWriter != nil {
				ic.connWriter.DisconnectClient(conn.ClientID)
			}
		}
	}
}

func (ic *IdleChecker) Stop() {
	ic.stopOnce.Do(func() {
		if ic.ticker != nil {
			ic.ticker.Stop()
		}
		close(ic.done)
		ic.logger.Info("Idle checker stopped")
	})
}
