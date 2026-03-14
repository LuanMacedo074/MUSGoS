package outbound_test

import (
	"testing"

	"fsos-server/internal/adapters/outbound"
)

const testKeyPrefix = "musgo_test"

// Redis session store tests use the shared interface tests via
// memory_session_store_test.go. The Redis-specific behavior (key prefixing,
// TTL, pipeline batching) is verified through integration tests
// only when a Redis instance is available.
//
// NewRedisSessionStoreWithClient accepts redis.Cmdable for testability.

func TestRedisSessionStore_WithClientConstructor(t *testing.T) {
	// Verify the WithClient constructor exists and returns a valid store.
	// We can't call methods without a real redis.Cmdable, but this
	// ensures the API compiles and the constructor doesn't panic.
	store := outbound.NewRedisSessionStoreWithClient(nil, testKeyPrefix, 3600)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}
