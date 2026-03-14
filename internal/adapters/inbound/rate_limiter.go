package inbound

import (
	"sync"
	"time"
)

type RateLimiterConfig struct {
	MaxRequests int
	Window      time.Duration
}

type clientBucket struct {
	tokens    float64
	lastCheck time.Time
}

type RateLimiter struct {
	mu              sync.Mutex
	maxTokens       float64
	refillRate      float64 // tokens per second
	clients         map[string]*clientBucket
	cleanupInterval time.Duration
	done            chan struct{}
}

func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	cleanupInterval := cfg.Window
	if cleanupInterval < 60*time.Second {
		cleanupInterval = 60 * time.Second
	}

	rl := &RateLimiter{
		maxTokens:       float64(cfg.MaxRequests),
		refillRate:      float64(cfg.MaxRequests) / cfg.Window.Seconds(),
		clients:         make(map[string]*clientBucket),
		cleanupInterval: cleanupInterval,
		done:            make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	cb, ok := rl.clients[key]
	if !ok {
		cb = &clientBucket{tokens: rl.maxTokens, lastCheck: now}
		rl.clients[key] = cb
	}

	elapsed := now.Sub(cb.lastCheck).Seconds()
	cb.tokens += elapsed * rl.refillRate
	if cb.tokens > rl.maxTokens {
		cb.tokens = rl.maxTokens
	}
	cb.lastCheck = now

	if cb.tokens < 1 {
		return false
	}

	cb.tokens--
	return true
}

func (rl *RateLimiter) Remove(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.clients, key)
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			for key, cb := range rl.clients {
				elapsed := time.Since(cb.lastCheck).Seconds()
				refilled := cb.tokens + elapsed*rl.refillRate
				if refilled >= rl.maxTokens {
					delete(rl.clients, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.done)
}
