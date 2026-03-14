package inbound

import (
	"sync"
	"time"

	"fsos-server/internal/domain/ports"
)

type TimerManager struct {
	mu           sync.Mutex
	serverTimer  *time.Timer
	userTimers   map[string]*time.Timer
	logger       ports.Logger
	sessionStore ports.SessionStore
	connWriter   ports.ConnectionWriter
	shutdownFn   func()
}

func NewTimerManager(sessionStore ports.SessionStore, connWriter ports.ConnectionWriter, logger ports.Logger, shutdownFn func()) *TimerManager {
	return &TimerManager{
		userTimers:   make(map[string]*time.Timer),
		logger:       logger,
		sessionStore: sessionStore,
		connWriter:   connWriter,
		shutdownFn:   shutdownFn,
	}
}

func (tm *TimerManager) SetServerKillTimer(minutes int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.serverTimer != nil {
		tm.serverTimer.Stop()
	}

	tm.logger.Info("Server kill timer set", map[string]interface{}{
		"minutes": minutes,
	})

	tm.serverTimer = time.AfterFunc(time.Duration(minutes)*time.Minute, func() {
		tm.logger.Info("Server kill timer fired, shutting down...")
		tm.shutdownFn()
	})
}

func (tm *TimerManager) CancelServerKillTimer() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.serverTimer != nil {
		tm.serverTimer.Stop()
		tm.serverTimer = nil
		tm.logger.Info("Server kill timer cancelled")
	}
}

func (tm *TimerManager) SetUserKillTimer(clientID string, minutes int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if existing, ok := tm.userTimers[clientID]; ok {
		existing.Stop()
	}

	tm.logger.Info("User kill timer set", map[string]interface{}{
		"clientID": clientID,
		"minutes":  minutes,
	})

	tm.userTimers[clientID] = time.AfterFunc(time.Duration(minutes)*time.Minute, func() {
		tm.mu.Lock()
		delete(tm.userTimers, clientID)
		tm.mu.Unlock()

		tm.logger.Info("User kill timer fired", map[string]interface{}{
			"clientID": clientID,
		})
		tm.sessionStore.LeaveAllRooms(clientID)
		tm.sessionStore.UnregisterConnection(clientID)
		if tm.connWriter != nil {
			tm.connWriter.DisconnectClient(clientID)
		}
	})
}

func (tm *TimerManager) CancelUserKillTimer(clientID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if timer, ok := tm.userTimers[clientID]; ok {
		timer.Stop()
		delete(tm.userTimers, clientID)
		tm.logger.Info("User kill timer cancelled", map[string]interface{}{
			"clientID": clientID,
		})
	}
}

func (tm *TimerManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.serverTimer != nil {
		tm.serverTimer.Stop()
		tm.serverTimer = nil
	}

	for id, timer := range tm.userTimers {
		timer.Stop()
		delete(tm.userTimers, id)
	}
}
