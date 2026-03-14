package factory_test

import (
	"testing"

	"fsos-server/internal/config"
	"fsos-server/internal/factory"
)

func TestNewCache_Memory(t *testing.T) {
	cache, err := factory.NewCache("memory", config.RedisConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("cache should not be nil")
	}
	cache.Close()
}

func TestNewCache_Unknown(t *testing.T) {
	_, err := factory.NewCache("memcached", config.RedisConfig{})
	if err == nil {
		t.Error("expected error for unknown cache type")
	}
}
