package outbound_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestMemoryCache_OperationLock(t *testing.T) {
	cache := outbound.NewMemoryCache()
	t.Cleanup(func() { cache.Close() })
	testCacheOperationLock(t, cache, cache)
}

func testCacheOperationLock(t *testing.T, first, second ports.Cache) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	release, err := first.AcquireLock(ctx, "operation")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { release.Release() })
	other, err := second.AcquireLock(ctx, "other-operation")
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Release(); err != nil {
		t.Fatal(err)
	}
	waitCtx, stop := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stop()
	if unlock, err := second.AcquireLock(waitCtx, "operation"); !errors.Is(err, context.DeadlineExceeded) {
		if unlock != nil {
			unlock.Release()
		}
		t.Fatalf("overlapping operation entered: %v", err)
	}
	if err := release.Release(); err != nil {
		t.Fatal(err)
	}
	next, err := second.AcquireLock(ctx, "operation")
	if err != nil {
		t.Fatal(err)
	}
	// Releasing an already completed operation cannot unlock the next owner.
	if err := release.Release(); err != nil {
		t.Fatal(err)
	}
	waitCtx2, stop2 := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stop2()
	if unlock, err := first.AcquireLock(waitCtx2, "operation"); !errors.Is(err, context.DeadlineExceeded) {
		if unlock != nil {
			unlock.Release()
		}
		t.Fatalf("old release unlocked the new owner: %v", err)
	}
	if err := next.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryCache_OperationLockRequiresADeadline(t *testing.T) {
	cache := outbound.NewMemoryCache()
	t.Cleanup(func() { cache.Close() })
	lease, err := cache.AcquireLock(context.Background(), "operation")
	if err == nil {
		lease.Release()
		t.Fatal("a lock without a deadline can never be reclaimed")
	}
}
