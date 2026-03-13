package outbound

import (
	"fmt"
	"sync"

	"fsos-server/internal/domain/ports"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitSubscription struct {
	channel *amqp.Channel
	tag     string
}

type RabbitMQQueue struct {
	conn     *amqp.Connection
	pubChan  *amqp.Channel
	exchange string

	mu      sync.Mutex
	subs    map[string]*rabbitSubscription
	closed  bool
}

func NewRabbitMQQueue(host, port, user, password, vhost, exchange string) (*RabbitMQQueue, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/%s", user, password, host, port, vhost)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ at %s:%s: %w", host, port, err)
	}

	pubChan, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	err = pubChan.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
	if err != nil {
		pubChan.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange %q: %w", exchange, err)
	}

	return &RabbitMQQueue{
		conn:     conn,
		pubChan:  pubChan,
		exchange: exchange,
		subs:     make(map[string]*rabbitSubscription),
	}, nil
}

func (q *RabbitMQQueue) Publish(topic string, payload []byte) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	q.mu.Unlock()

	return q.pubChan.Publish(q.exchange, topic, false, false, amqp.Publishing{
		ContentType: "application/octet-stream",
		Body:        payload,
	})
}

func (q *RabbitMQQueue) Subscribe(topic string, handler ports.QueueSubscriber) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	ch, err := q.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	tmpQueue, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		ch.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	err = ch.QueueBind(tmpQueue.Name, topic, q.exchange, false, nil)
	if err != nil {
		ch.Close()
		return fmt.Errorf("failed to bind queue to topic %q: %w", topic, err)
	}

	deliveries, err := ch.Consume(tmpQueue.Name, "", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	q.subs[topic] = &rabbitSubscription{channel: ch, tag: ""}

	go func() {
		for d := range deliveries {
			handler(ports.QueueMessage{Topic: topic, Payload: d.Body})
		}
	}()

	return nil
}

func (q *RabbitMQQueue) Unsubscribe(topic string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	if sub, ok := q.subs[topic]; ok {
		sub.channel.Close()
		delete(q.subs, topic)
	}
	return nil
}

func (q *RabbitMQQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.closed = true
	for topic, sub := range q.subs {
		sub.channel.Close()
		delete(q.subs, topic)
	}
	// Ignore error — channel may already be closed by broker disconnect
	_ = q.pubChan.Close()
	return q.conn.Close()
}
