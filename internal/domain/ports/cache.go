package ports

import (
	"context"
	"time"
)

// CacheLock owns an operation scope. Context is canceled if ownership is lost.
// Release ends ownership and is safe to call more than once.
type CacheLock struct {
	Context context.Context
	Release func() error
}

type Cache interface {
	// AcquireLock serializes operations sharing a key until release is called.
	// The caller must stop work when the lock's Context ends; ctx needs a deadline.
	// Release is idempotent and must run on every exit, including errors.
	AcquireLock(ctx context.Context, key string) (*CacheLock, error)
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	// Increment atomically adds one to an integer value, starting a missing key at one.
	// It preserves an existing expiry and rejects non-integers and overflow.
	Increment(key string) (int64, error)
	Delete(key string) error
	Exists(key string) (bool, error)
	SetAdd(key, member string) error
	// SetAddOnce atomically adds a member and reports whether it was new.
	SetAddOnce(key, member string) (bool, error)
	SetRemove(key, member string) error
	SetMembers(key string) ([]string, error)
	SetIsMember(key, member string) (bool, error)
	Close() error
}
