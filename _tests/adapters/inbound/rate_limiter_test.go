package inbound_test

import (
	"testing"
	"time"

	"fsos-server/internal/adapters/inbound"
)

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := inbound.NewRateLimiter(inbound.RateLimiterConfig{
		MaxRequests: 5,
		Window:      time.Second,
	})
	defer rl.Stop()

	for i := 0; i < 5; i++ {
		if !rl.Allow("client1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := inbound.NewRateLimiter(inbound.RateLimiterConfig{
		MaxRequests: 3,
		Window:      time.Second,
	})
	defer rl.Stop()

	for i := 0; i < 3; i++ {
		rl.Allow("client1")
	}

	if rl.Allow("client1") {
		t.Fatal("4th request should be blocked")
	}
}

func TestRateLimiter_SeparateClients(t *testing.T) {
	rl := inbound.NewRateLimiter(inbound.RateLimiterConfig{
		MaxRequests: 2,
		Window:      time.Second,
	})
	defer rl.Stop()

	rl.Allow("client1")
	rl.Allow("client1")

	if rl.Allow("client1") {
		t.Fatal("client1 should be blocked")
	}

	if !rl.Allow("client2") {
		t.Fatal("client2 should be allowed")
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	rl := inbound.NewRateLimiter(inbound.RateLimiterConfig{
		MaxRequests: 2,
		Window:      100 * time.Millisecond,
	})
	defer rl.Stop()

	rl.Allow("client1")
	rl.Allow("client1")

	if rl.Allow("client1") {
		t.Fatal("should be blocked before tokens refill")
	}

	time.Sleep(150 * time.Millisecond)

	if !rl.Allow("client1") {
		t.Fatal("should be allowed after tokens refill")
	}
}

func TestRateLimiter_Remove(t *testing.T) {
	rl := inbound.NewRateLimiter(inbound.RateLimiterConfig{
		MaxRequests: 1,
		Window:      time.Second,
	})
	defer rl.Stop()

	rl.Allow("client1")
	if rl.Allow("client1") {
		t.Fatal("should be blocked")
	}

	rl.Remove("client1")

	if !rl.Allow("client1") {
		t.Fatal("should be allowed after Remove")
	}
}
