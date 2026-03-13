package outbound_test

import (
	"sync"
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestMemoryQueue_PublishSubscribe(t *testing.T) {
	q := outbound.NewMemoryQueue()
	defer q.Close()

	var received ports.QueueMessage
	var wg sync.WaitGroup
	wg.Add(1)

	q.Subscribe("test.topic", func(msg ports.QueueMessage) {
		received = msg
		wg.Done()
	})

	q.Publish("test.topic", []byte("hello"))

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}

	if received.Topic != "test.topic" {
		t.Errorf("topic = %q, want %q", received.Topic, "test.topic")
	}
	if string(received.Payload) != "hello" {
		t.Errorf("payload = %q, want %q", received.Payload, "hello")
	}
}

func TestMemoryQueue_MultipleSubscribers(t *testing.T) {
	q := outbound.NewMemoryQueue()
	defer q.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	count := 0
	var mu sync.Mutex

	handler := func(msg ports.QueueMessage) {
		mu.Lock()
		count++
		mu.Unlock()
		wg.Done()
	}

	q.Subscribe("events", handler)
	q.Subscribe("events", handler)

	q.Publish("events", []byte("data"))

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for messages")
	}

	mu.Lock()
	if count != 2 {
		t.Errorf("received %d messages, want 2", count)
	}
	mu.Unlock()
}

func TestMemoryQueue_Unsubscribe(t *testing.T) {
	q := outbound.NewMemoryQueue()
	defer q.Close()

	called := false
	q.Subscribe("topic", func(msg ports.QueueMessage) {
		called = true
	})

	q.Unsubscribe("topic")
	q.Publish("topic", []byte("data"))

	time.Sleep(50 * time.Millisecond)
	if called {
		t.Error("handler should not be called after unsubscribe")
	}
}

func TestMemoryQueue_PublishNoSubscribers(t *testing.T) {
	q := outbound.NewMemoryQueue()
	defer q.Close()

	err := q.Publish("nobody.listening", []byte("data"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemoryQueue_ClosePreventsFurtherOps(t *testing.T) {
	q := outbound.NewMemoryQueue()
	q.Close()

	if err := q.Publish("topic", []byte("data")); err != outbound.ErrQueueClosed {
		t.Errorf("Publish after Close: err = %v, want ErrQueueClosed", err)
	}
	if err := q.Subscribe("topic", func(msg ports.QueueMessage) {}); err != outbound.ErrQueueClosed {
		t.Errorf("Subscribe after Close: err = %v, want ErrQueueClosed", err)
	}
	if err := q.Unsubscribe("topic"); err != outbound.ErrQueueClosed {
		t.Errorf("Unsubscribe after Close: err = %v, want ErrQueueClosed", err)
	}
}
