package outbound_test

import (
	"sync"
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestMemoryCache_Increment(t *testing.T) {
	cache := outbound.NewMemoryCache()
	t.Cleanup(func() { cache.Close() })
	testCacheIncrement(t, cache)
}

func testCacheIncrement(t *testing.T, cache ports.Cache) {
	t.Helper()
	t.Run("concurrent callers get distinct revisions", func(t *testing.T) {
		results := make(chan int64, 100)
		errors := make(chan error, 100)
		var workers sync.WaitGroup
		for range 100 {
			workers.Go(func() {
				value, err := cache.Increment("counter")
				results <- value
				errors <- err
			})
		}
		workers.Wait()
		close(results)
		close(errors)
		for err := range errors {
			if err != nil {
				t.Fatal(err)
			}
		}
		seen := make(map[int64]bool)
		for value := range results {
			if value < 1 || value > 100 || seen[value] {
				t.Fatalf("invalid or repeated revision: %d", value)
			}
			seen[value] = true
		}
		value, err := cache.Get("counter")
		if err != nil {
			t.Fatal(err)
		}
		if string(value) != "100" {
			t.Fatalf("counter = %q, want 100", value)
		}
	})
	t.Run("invalid counters remain unchanged", func(t *testing.T) {
		for _, initial := range []string{"not-an-integer", "9223372036854775807"} {
			if err := cache.Set("invalid", []byte(initial), 0); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.Increment("invalid"); err == nil {
				t.Fatalf("increment accepted %q", initial)
			}
			value, err := cache.Get("invalid")
			if err != nil {
				t.Fatal(err)
			}
			if string(value) != initial {
				t.Fatalf("counter = %q, want %q", value, initial)
			}
		}
	})
	t.Run("existing expiry is preserved", func(t *testing.T) {
		if err := cache.Set("expiring", []byte("7"), 100*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		if value, err := cache.Increment("expiring"); err != nil || value != 8 {
			t.Fatalf("increment = %d, %v; want 8", value, err)
		}
		time.Sleep(150 * time.Millisecond)
		value, err := cache.Get("expiring")
		if err != nil {
			t.Fatal(err)
		}
		if value != nil {
			t.Fatalf("counter did not expire: %q", value)
		}
		if value, err := cache.Increment("expiring"); err != nil || value != 1 {
			t.Fatalf("expired increment = %d, %v; want 1", value, err)
		}
	})
}
