package outbound

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

func (e *cacheEntry) expired() bool {
	if e.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.expiresAt)
}

type MemoryCache struct {
	mu            sync.RWMutex
	entries       map[string]*cacheEntry
	closed        bool
	writeCount    int
	sweepInterval int
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		entries:       make(map[string]*cacheEntry),
		sweepInterval: 1000,
	}
}

func (c *MemoryCache) Get(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || entry.expired() {
		return nil, nil
	}

	copied := make([]byte, len(entry.value))
	copy(copied, entry.value)
	return copied, nil
}

func (c *MemoryCache) Set(key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	copied := make([]byte, len(value))
	copy(copied, value)

	entry := &cacheEntry{value: copied}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	c.entries[key] = entry

	c.writeCount++
	if c.writeCount >= c.sweepInterval {
		c.writeCount = 0
		c.sweep()
	}

	return nil
}

func (c *MemoryCache) sweep() {
	for k, entry := range c.entries {
		if entry.expired() {
			delete(c.entries, k)
		}
	}
}

func (c *MemoryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	return nil
}

func (c *MemoryCache) Exists(key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || entry.expired() {
		return false, nil
	}
	return true, nil
}

func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = nil
	c.closed = true
	return nil
}
