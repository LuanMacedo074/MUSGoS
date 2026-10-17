package outbound

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fsos-server/internal/domain/ports"

	"github.com/google/uuid"
)

// Release only our lease: a delayed release must not delete a later owner's lock.
const releaseCacheLock = `if redis.call('get', KEYS[1]) == ARGV[1] then
 return redis.call('del', KEYS[1])
end
return 0`

const renewCacheLock = `if redis.call('get', KEYS[1]) == ARGV[1] then
 return redis.call('pexpire', KEYS[1], 30000)
end
return 0`

// Keep the process's execution scope even if its backend lease is lost. A
// blocked native callback must finish before another local execution can enter.
var redisExecutionLocks = NewMemoryCache()

func (c *RedisCache) AcquireLock(ctx context.Context, key string) (*ports.CacheLock, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, fmt.Errorf("cache lock requires a deadline")
	}
	local, err := redisExecutionLocks.AcquireLock(ctx, c.key(key))
	if err != nil {
		return nil, err
	}
	retained := false
	defer func() {
		if !retained {
			local.Release()
		}
	}()
	token := uuid.NewString()
	lockKey := c.key("lock:" + key)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Acquisition is deadline-bound, but ownership lasts through actual exit.
		// Native callbacks may remain active after Lua's deadline has elapsed.
		acquired, err := c.client.SetNX(ctx, lockKey, token, 30*time.Second).Result()
		if err != nil {
			return nil, err
		}
		if acquired {
			operationCtx, cancel := context.WithCancel(local.Context)
			releaseBackend := c.maintainLock(lockKey, token, cancel)
			release := func() error {
				err := releaseBackend()
				local.Release()
				return err
			}
			if err := ctx.Err(); err != nil {
				release()
				return nil, err
			}
			retained = true
			return &ports.CacheLock{Context: operationCtx, Release: release}, nil
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// Renew independently of the Lua deadline until its execution actually exits.
func (c *RedisCache) maintainLock(key, token string, cancelOperation context.CancelFunc) func() error {
	stop := make(chan struct{})
	done := make(chan struct{})
	var renewalErr error
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				renewed, err := c.client.Eval(ctx, renewCacheLock, []string{key}, token).Int64()
				if err != nil || renewed == 0 {
					renewalErr = fmt.Errorf("cache lock renewal failed: renewed=%d error=%v", renewed, err)
					cancelOperation()
				}
				cancel()
			}
		}
	}()
	var once sync.Once
	var releaseErr error
	return func() error {
		once.Do(func() {
			cancelOperation()
			close(stop)
			<-done
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			releaseErr = c.client.Eval(ctx, releaseCacheLock, []string{key}, token).Err()
			if releaseErr == nil {
				releaseErr = renewalErr
			}
		})
		return releaseErr
	}
}
