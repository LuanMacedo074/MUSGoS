package outbound_test

import (
	"testing"

	"fsos-server/internal/adapters/outbound"
)

const testCacheKeyPrefix = "musgoc_test"

// Redis cache tests verify construction. The cache interface behavior
// is covered by memory_cache_test.go through the same ports.Cache
// interface. NewRedisCacheWithClient accepts redis.Cmdable for testability.

func TestRedisCache_WithClientConstructor(t *testing.T) {
	c := outbound.NewRedisCacheWithClient(nil, testCacheKeyPrefix)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
}
