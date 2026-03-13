package outbound_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"

	"github.com/redis/go-redis/v9"
)

const testQueueKeyPrefix = "musgoq_test"

func skipIfNoRedisForQueue(t *testing.T) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
}

func newRedisTestQueue(t *testing.T) *outbound.RedisQueue {
	t.Helper()
	skipIfNoRedisForQueue(t)

	q, err := outbound.NewRedisQueue("localhost:6379", "", 0, testQueueKeyPrefix)
	if err != nil {
		t.Fatalf("failed to create redis queue: %v", err)
	}

	t.Cleanup(func() {
		q.Close()
		client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		defer client.Close()
		iter := client.Scan(context.Background(), 0, testQueueKeyPrefix+":*", 0).Iterator()
		for iter.Next(context.Background()) {
			client.Del(context.Background(), iter.Val())
		}
	})

	return q
}

func TestRedisQueue_PublishSubscribe(t *testing.T) {
	q := newRedisTestQueue(t)

	var received ports.QueueMessage
	var wg sync.WaitGroup
	wg.Add(1)

	q.Subscribe("test.topic", func(msg ports.QueueMessage) {
		received = msg
		wg.Done()
	})

	// Give consumer goroutine time to start BRPOP
	time.Sleep(100 * time.Millisecond)

	q.Publish("test.topic", []byte("hello-redis"))

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	if received.Topic != "test.topic" {
		t.Errorf("topic = %q, want %q", received.Topic, "test.topic")
	}
	if string(received.Payload) != "hello-redis" {
		t.Errorf("payload = %q, want %q", received.Payload, "hello-redis")
	}
}

func TestRedisQueue_Unsubscribe(t *testing.T) {
	q := newRedisTestQueue(t)

	called := false
	q.Subscribe("topic", func(msg ports.QueueMessage) {
		called = true
	})

	q.Unsubscribe("topic")
	time.Sleep(50 * time.Millisecond)

	q.Publish("topic", []byte("data"))
	time.Sleep(200 * time.Millisecond)

	if called {
		t.Error("handler should not be called after unsubscribe")
	}
}
