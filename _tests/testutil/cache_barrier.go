package testutil

import "musgos/internal/domain/ports"

// ScopedLockBarrier pauses the "after-scope" sentinel read so a test can observe
// whether a completed Lua lock scope released its lease before the script exits.
type ScopedLockBarrier struct {
	ports.Cache
	After, Resume chan struct{}
}

func (c *ScopedLockBarrier) Get(key string) ([]byte, error) {
	if key == "after-scope" {
		close(c.After)
		<-c.Resume
	}
	return c.Cache.Get(key)
}
