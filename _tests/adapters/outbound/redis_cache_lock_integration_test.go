//go:build integration

package outbound_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestIntegrationRedisCache_OperationLock(t *testing.T) {
	first := newRedisCache(t)
	second, err := outbound.NewRedisCache(redisAddr(), redisPassword(), redisDB(), "musgotest:"+runNonce+":cache:"+safeName(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { second.Close() })
	testCacheOperationLock(t, first, second)
}

func TestIntegrationRedisCache_LockCoordinatesSeparateProcesses(t *testing.T) {
	prefix := "musgotest:" + runNonce + ":remote-lock:" + uuid.NewString()
	cache, err := outbound.NewRedisCache(redisAddr(), redisPassword(), redisDB(), prefix)
	if err != nil {
		t.Skipf("redis not reachable at %s (run 'make thirdparties-up'): %v", redisAddr(), err)
	}
	defer cache.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lease, err := cache.AcquireLock(ctx, "operation")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	child := exec.Command(os.Args[0], "-test.run=^TestIntegrationRedisCache_RemoteLockClient$")
	child.Env = append(os.Environ(), "CACHE_LOCK_TEST_PREFIX="+prefix)
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("remote lock contender: %v\n%s", err, output)
	}
}

func TestIntegrationRedisCache_RemoteLockClient(t *testing.T) {
	prefix := os.Getenv("CACHE_LOCK_TEST_PREFIX")
	if prefix == "" {
		t.Skip("subprocess fixture")
	}
	cache, err := outbound.NewRedisCache(redisAddr(), redisPassword(), redisDB(), prefix)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if lease, err := cache.AcquireLock(ctx, "operation"); !errors.Is(err, context.DeadlineExceeded) {
		if lease != nil {
			lease.Release()
		}
		t.Fatalf("remote operation entered an owned scope: %v", err)
	}
}

type nativeCacheReadBarrier struct {
	ports.Cache
	waiting, release chan struct{}
}

func (c *nativeCacheReadBarrier) Get(key string) ([]byte, error) {
	close(c.waiting)
	<-c.release
	return c.Cache.Get(key)
}

type nativeCacheWriteBarrier struct {
	ports.Cache
	waiting, release chan struct{}
}

func (c *nativeCacheWriteBarrier) Set(key string, value []byte, ttl time.Duration) error {
	close(c.waiting)
	<-c.release
	return c.Cache.Set(key, value, ttl)
}

// releaseBarrierOnCleanup unblocks a paused cache barrier when a test fails
// before it got there, so the execution under test can never outlive the test.
func releaseBarrierOnCleanup(t *testing.T, release chan struct{}) {
	t.Helper()
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
}

func awaitCacheBarrier(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("operation did not reach its cache boundary")
	}
}

func TestIntegrationRedisCache_LuaLockSurvivesABlockedNativeCallback(t *testing.T) {
	cache := newRedisCache(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "operation.lua"), []byte("mus.cache.lock('delayed')\nmus.cache.get('pause')"), 0600); err != nil {
		t.Fatal(err)
	}
	barrier := &nativeCacheReadBarrier{Cache: cache, waiting: make(chan struct{}), release: make(chan struct{})}
	releaseBarrierOnCleanup(t, barrier.release)
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 1, nil, nil, nil, nil, nil, barrier, nil)
	done := make(chan error, 1)
	go func() { _, err := engine.Execute(&ports.ScriptMessage{Subject: "operation"}); done <- err }()
	awaitCacheBarrier(t, barrier.waiting)
	// The first lease expired at the one-second Lua deadline plus 30 seconds.
	// The synchronous Go callback remains active beyond that entire interval.
	time.Sleep(32 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if release, err := cache.AcquireLock(ctx, "delayed"); !errors.Is(err, context.DeadlineExceeded) {
		if release != nil {
			release.Release()
		}
		close(barrier.release)
		<-done
		t.Fatalf("unfinished native callback lost ownership: %v", err)
	}
	close(barrier.release)
	if err := <-done; err == nil {
		t.Fatal("Lua execution should report its expired deadline")
	}
	nextCtx, nextCancel := context.WithTimeout(context.Background(), time.Second)
	defer nextCancel()
	release, err := cache.AcquireLock(nextCtx, "delayed")
	if err != nil {
		t.Fatalf("exited execution retained its lock: %v", err)
	}
	if err := release.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationRedisCache_RenewalFailureDoesNotOverlapANativeMutation(t *testing.T) {
	admin := redis.NewClient(&redis.Options{Addr: redisAddr(), Password: redisPassword(), DB: redisDB()})
	t.Cleanup(func() { admin.Close() })
	ctx := context.Background()
	if err := admin.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable at %s (run 'make thirdparties-up'): %v", redisAddr(), err)
	}
	user := "lock-" + uuid.NewString()
	password := uuid.NewString()
	if err := admin.Do(ctx, "ACL", "SETUSER", user, "on", ">"+password, "~*", "+@all", "-pexpire").Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Do(ctx, "ACL", "DELUSER", user) })
	ownerClient := redis.NewClient(&redis.Options{Addr: redisAddr(), Username: user, Password: password, DB: redisDB()})
	t.Cleanup(func() { ownerClient.Close() })
	if who, err := ownerClient.Do(ctx, "ACL", "WHOAMI").Text(); err != nil || who != user {
		t.Fatalf("renewal-failure client did not use its restricted user: %q, %v", who, err)
	}
	prefix := "musgotest:" + runNonce + ":lease-failure:" + uuid.NewString()
	owner := outbound.NewRedisCacheWithClient(ownerClient, prefix)
	peer := outbound.NewRedisCacheWithClient(admin, prefix)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "operation.lua"), []byte("mus.cache.lock('mutation')\nmus.cache.set('state', 'finished')"), 0600); err != nil {
		t.Fatal(err)
	}
	barrier := &nativeCacheWriteBarrier{Cache: owner, waiting: make(chan struct{}), release: make(chan struct{})}
	releaseBarrierOnCleanup(t, barrier.release)
	engine := outbound.NewLuaScriptEngine(dir, &testutil.MockLogger{}, 50, nil, nil, nil, nil, nil, barrier, nil)
	done := make(chan error, 1)
	go func() { _, err := engine.Execute(&ports.ScriptMessage{Subject: "operation"}); done <- err }()
	awaitCacheBarrier(t, barrier.waiting)
	// Redis refuses renewal while the native mutation remains blocked; the
	// thirty-second backend lease expires before that callback can return.
	time.Sleep(32 * time.Second)
	waitCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if lease, err := peer.AcquireLock(waitCtx, "mutation"); !errors.Is(err, context.DeadlineExceeded) {
		if lease != nil {
			lease.Release()
		}
		close(barrier.release)
		<-done
		t.Fatalf("renewal failure exposed an unfinished mutation: %v", err)
	}
	close(barrier.release)
	if err := <-done; err == nil {
		t.Fatal("renewal failure should cancel the Lua execution")
	}
	nextCtx, stop := context.WithTimeout(ctx, time.Second)
	defer stop()
	lease, err := peer.AcquireLock(nextCtx, "mutation")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	value, err := peer.Get("state")
	if err != nil || string(value) != "finished" {
		t.Fatalf("completed native mutation = %q, %v", value, err)
	}
}
