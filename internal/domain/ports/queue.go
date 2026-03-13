package ports

// QueueMessage is the envelope published through the queue.
type QueueMessage struct {
	Topic   string
	Payload []byte
}

// QueueSubscriber handles messages received from a queue topic.
type QueueSubscriber func(msg QueueMessage)

// QueuePublisher publishes messages to topics.
type QueuePublisher interface {
	Publish(topic string, payload []byte) error
	Close() error
}

// QueueConsumer receives messages from topics.
type QueueConsumer interface {
	Subscribe(topic string, handler QueueSubscriber) error
	Unsubscribe(topic string) error
	Close() error
}

// MessageQueue combines publisher and consumer capabilities.
type MessageQueue interface {
	QueuePublisher
	QueueConsumer
}
