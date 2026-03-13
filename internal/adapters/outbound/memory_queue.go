package outbound

import (
	"sync"

	"fsos-server/internal/domain/ports"
)

type MemoryQueue struct {
	mu          sync.RWMutex
	subscribers map[string][]ports.QueueSubscriber
	closed      bool
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		subscribers: make(map[string][]ports.QueueSubscriber),
	}
}

func (q *MemoryQueue) Publish(topic string, payload []byte) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return ErrQueueClosed
	}

	subs := q.subscribers[topic]
	for _, handler := range subs {
		handler := handler
		go handler(ports.QueueMessage{Topic: topic, Payload: payload})
	}
	return nil
}

func (q *MemoryQueue) Subscribe(topic string, handler ports.QueueSubscriber) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	q.subscribers[topic] = append(q.subscribers[topic], handler)
	return nil
}

func (q *MemoryQueue) Unsubscribe(topic string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	delete(q.subscribers, topic)
	return nil
}

func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.closed = true
	q.subscribers = nil
	return nil
}
