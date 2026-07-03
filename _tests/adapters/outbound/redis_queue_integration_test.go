//go:build integration

package outbound_test

import (
	"sync"
	"testing"
	"time"

	"fsos-server/internal/domain/ports"
)

func TestIntegrationRedisQueue_PublishSubscribe(t *testing.T) {
	q := newRedisQueue(t)

	var received ports.QueueMessage
	var wg sync.WaitGroup
	wg.Add(1)
	mustNoErr(t, q.Subscribe("test.topic", func(msg ports.QueueMessage) {
		received = msg
		wg.Done()
	}))
	// Give the consumer goroutine a moment to enter its BRPOP before publishing.
	time.Sleep(100 * time.Millisecond)
	mustNoErr(t, q.Publish("test.topic", []byte("hello")))

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second): // BRPOP polls at 1s granularity
		t.Fatal("timed out waiting for message")
	}

	if received.Topic != "test.topic" {
		t.Errorf("topic = %q, want %q", received.Topic, "test.topic")
	}
	if string(received.Payload) != "hello" {
		t.Errorf("payload = %q, want %q", received.Payload, "hello")
	}
}

func TestIntegrationRedisQueue_Unsubscribe(t *testing.T) {
	q := newRedisQueue(t)

	var mu sync.Mutex
	called := false
	mustNoErr(t, q.Subscribe("topic", func(msg ports.QueueMessage) {
		mu.Lock()
		called = true
		mu.Unlock()
	}))
	mustNoErr(t, q.Unsubscribe("topic"))

	mustNoErr(t, q.Publish("topic", []byte("data")))
	time.Sleep(2 * time.Second) // longer than the 1s poll, so a live consumer would fire

	mu.Lock()
	defer mu.Unlock()
	if called {
		t.Error("handler should not be called after unsubscribe")
	}
}
