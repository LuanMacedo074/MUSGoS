package outbound

import (
	"context"
	"fmt"
	"sync"

	"fsos-server/internal/domain/ports"
)

type cacheLock struct {
	gate  chan struct{}
	users int
}

func (c *MemoryCache) AcquireLock(ctx context.Context, key string) (*ports.CacheLock, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, fmt.Errorf("cache lock requires a deadline")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	lock := c.locks[key]
	if lock == nil {
		lock = &cacheLock{gate: make(chan struct{}, 1)}
		lock.gate <- struct{}{}
		c.locks[key] = lock
	}
	lock.users++
	c.mu.Unlock()
	unref := func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		lock.users--
		if lock.users == 0 {
			delete(c.locks, key)
		}
	}
	select {
	case <-ctx.Done():
		unref()
		return nil, ctx.Err()
	case <-lock.gate:
		operationCtx, cancel := context.WithCancel(ctx)
		var once sync.Once
		release := func() error {
			once.Do(func() { cancel(); lock.gate <- struct{}{}; unref() })
			return nil
		}
		if err := ctx.Err(); err != nil {
			release()
			return nil, err
		}
		return &ports.CacheLock{Context: operationCtx, Release: release}, nil
	}
}
