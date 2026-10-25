package outbound_test

import (
	"sync"
	"testing"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestMemoryCache_SetAddOnce(t *testing.T) {
	cache := outbound.NewMemoryCache()
	t.Cleanup(func() { cache.Close() })
	testCacheSetAddOnce(t, cache)
}

func testCacheSetAddOnce(t *testing.T, cache ports.Cache) {
	t.Helper()
	t.Run("reports only the first insertion", func(t *testing.T) {
		added, err := cache.SetAddOnce("ledger", "action-1")
		if err != nil || !added {
			t.Fatalf("first SetAddOnce = %t, %v; want true", added, err)
		}
		added, err = cache.SetAddOnce("ledger", "action-1")
		if err != nil || added {
			t.Fatalf("repeated SetAddOnce = %t, %v; want false", added, err)
		}
		present, err := cache.SetIsMember("ledger", "action-1")
		if err != nil || !present {
			t.Fatalf("SetIsMember = %t, %v; want true", present, err)
		}
	})
	t.Run("concurrent callers admit one winner", func(t *testing.T) {
		results := make(chan bool, 100)
		errors := make(chan error, 100)
		var workers sync.WaitGroup
		for range 100 {
			workers.Go(func() {
				added, err := cache.SetAddOnce("contested", "action")
				results <- added
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
		winners := 0
		for added := range results {
			if added {
				winners++
			}
		}
		if winners != 1 {
			t.Fatalf("winners = %d, want 1", winners)
		}
	})
}
