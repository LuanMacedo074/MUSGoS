package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"fsos-server/internal/domain/ports"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client    redis.Cmdable
	closer    func() error
	keyPrefix string

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
	closed  bool
}

func NewRedisQueue(addr, password string, db int, keyPrefix string) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	return &RedisQueue{
		client:    client,
		closer:    client.Close,
		keyPrefix: keyPrefix,
		cancels:   make(map[string]context.CancelFunc),
	}, nil
}

// NewRedisQueueWithClient creates a RedisQueue using an existing redis.Cmdable (for testing).
func NewRedisQueueWithClient(client redis.Cmdable, keyPrefix string) *RedisQueue {
	return &RedisQueue{
		client:    client,
		closer:    func() error { return nil },
		keyPrefix: keyPrefix,
		cancels:   make(map[string]context.CancelFunc),
	}
}

func (q *RedisQueue) topicKey(topic string) string {
	return q.keyPrefix + ":" + topic
}

func (q *RedisQueue) Publish(topic string, payload []byte) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	q.mu.Unlock()

	msg := redisQueueMessage{Payload: payload}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal queue message: %w", err)
	}

	return q.client.LPush(context.Background(), q.topicKey(topic), data).Err()
}

func (q *RedisQueue) Subscribe(topic string, handler ports.QueueSubscriber) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	ctx, cancel := context.WithCancel(context.Background())
	q.cancels[topic] = cancel

	go q.consumeLoop(ctx, topic, handler)
	return nil
}

func (q *RedisQueue) consumeLoop(ctx context.Context, topic string, handler ports.QueueSubscriber) {
	key := q.topicKey(topic)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := q.client.BRPop(ctx, 1*time.Second, key).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		if len(result) < 2 {
			continue
		}

		var msg redisQueueMessage
		if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
			continue
		}

		handler(ports.QueueMessage{Topic: topic, Payload: msg.Payload})
	}
}

func (q *RedisQueue) Unsubscribe(topic string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	if cancel, ok := q.cancels[topic]; ok {
		cancel()
		delete(q.cancels, topic)
	}
	return nil
}

func (q *RedisQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.closed = true
	for topic, cancel := range q.cancels {
		cancel()
		delete(q.cancels, topic)
	}
	return q.closer()
}

type redisQueueMessage struct {
	Payload []byte `json:"payload"`
}
