package outbound_test

import (
	"testing"

	"fsos-server/internal/adapters/outbound"
)

const testQueueKeyPrefix = "musgoq_test"

// Redis queue tests verify construction. The queue interface behavior
// is covered by memory_queue_test.go through the same ports.MessageQueue
// interface. NewRedisQueueWithClient accepts redis.Cmdable for testability.

func TestRedisQueue_WithClientConstructor(t *testing.T) {
	q := outbound.NewRedisQueueWithClient(nil, testQueueKeyPrefix)
	if q == nil {
		t.Fatal("expected non-nil queue")
	}
}

func TestRedisQueue_CloseAfterConstruction(t *testing.T) {
	q := outbound.NewRedisQueueWithClient(nil, testQueueKeyPrefix)
	err := q.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestRedisQueue_PublishAfterClose(t *testing.T) {
	q := outbound.NewRedisQueueWithClient(nil, testQueueKeyPrefix)
	q.Close()

	err := q.Publish("test.topic", []byte("data"))
	if err != outbound.ErrQueueClosed {
		t.Errorf("Publish after Close: err = %v, want ErrQueueClosed", err)
	}
}
