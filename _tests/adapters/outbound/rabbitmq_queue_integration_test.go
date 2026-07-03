//go:build integration

package outbound_test

import (
	"sync"
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestIntegrationRabbitMQ_PublishSubscribe(t *testing.T) {
	q := newRabbitMQ(t)

	var received ports.QueueMessage
	var wg sync.WaitGroup
	wg.Add(1)
	mustNoErr(t, q.Subscribe("test.rmq", func(msg ports.QueueMessage) {
		received = msg
		wg.Done()
	}))
	// The binding is set up inside Subscribe; give the broker a moment before publishing.
	time.Sleep(200 * time.Millisecond)
	mustNoErr(t, q.Publish("test.rmq", []byte("hello")))

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	if received.Topic != "test.rmq" {
		t.Errorf("topic = %q, want %q", received.Topic, "test.rmq")
	}
	if string(received.Payload) != "hello" {
		t.Errorf("payload = %q, want %q", received.Payload, "hello")
	}
}

func TestIntegrationRabbitMQ_PublishAfterClose(t *testing.T) {
	q := newRabbitMQ(t)
	// Cleanup already registers a Close; calling it here too is safe/idempotent
	// enough for asserting the closed-queue behavior.
	mustNoErr(t, q.Close())

	if err := q.Publish("test.rmq", []byte("data")); err != outbound.ErrQueueClosed {
		t.Errorf("Publish after Close: err = %v, want ErrQueueClosed", err)
	}
}
