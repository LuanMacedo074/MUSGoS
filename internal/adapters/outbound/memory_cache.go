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
	sets          map[string]map[string]struct{}
	closed        bool
	writeCount    int
	sweepInterval int
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		entries:       make(map[string]*cacheEntry),
		sets:          make(map[string]map[string]struct{}),
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

func (c *MemoryCache) SetAdd(key, member string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, ok := c.sets[key]
	if !ok {
		s = make(map[string]struct{})
		c.sets[key] = s
	}
	s[member] = struct{}{}
	return nil
}

func (c *MemoryCache) SetRemove(key, member string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, ok := c.sets[key]
	if !ok {
		return nil
	}
	delete(s, member)
	if len(s) == 0 {
		delete(c.sets, key)
	}
	return nil
}

func (c *MemoryCache) SetMembers(key string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	s, ok := c.sets[key]
	if !ok {
		return []string{}, nil
	}
	members := make([]string, 0, len(s))
	for m := range s {
		members = append(members, m)
	}
	return members, nil
}

func (c *MemoryCache) SetIsMember(key, member string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	s, ok := c.sets[key]
	if !ok {
		return false, nil
	}
	_, exists := s[member]
	return exists, nil
}

func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = nil
	c.sets = nil
	c.closed = true
	return nil
}
