package outbound_test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"fsos-server/internal/adapters/outbound"
)

// The server shares a single Blowfish instance across every connection
// goroutine. This exercises that sharing: many goroutines Encrypt/Decrypt on one
// instance concurrently. Run with -race, it guards H1 (the shared cipher was
// mutated without locking). Each Encrypt/Decrypt resets state, so a correct
// round-trip must hold regardless of interleaving.
func TestBlowfish_ConcurrentSharedInstance(t *testing.T) {
	bf := outbound.NewBlowfish("sharedkey")
	bf.SetKey()

	const goroutines = 32
	const iterations = 200

	var wg sync.WaitGroup
	errCh := make(chan string, goroutines)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := []byte(fmt.Sprintf("goroutine-%02d-payload-abcdefgh", id))
			for i := 0; i < iterations; i++ {
				dec := bf.Decrypt(bf.Encrypt(msg))
				if !bytes.Equal(dec, msg) {
					errCh <- fmt.Sprintf("goroutine %d iter %d: round-trip mismatch", id, i)
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Error(e)
	}
}
