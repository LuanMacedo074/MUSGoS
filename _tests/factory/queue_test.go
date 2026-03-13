package factory_test

import (
	"testing"

	"fsos-server/internal/config"
	"fsos-server/internal/factory"
)

func TestNewMessageQueue_Memory(t *testing.T) {
	q, err := factory.NewMessageQueue("memory", config.RedisConfig{}, config.RabbitMQConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q == nil {
		t.Fatal("queue should not be nil")
	}
	q.Close()
}

func TestNewMessageQueue_Unknown(t *testing.T) {
	_, err := factory.NewMessageQueue("kafka", config.RedisConfig{}, config.RabbitMQConfig{})
	if err == nil {
		t.Error("expected error for unknown queue type")
	}
}

func TestNewMessageQueue_RedisInvalidDB(t *testing.T) {
	_, err := factory.NewMessageQueue("redis", config.RedisConfig{DB: "notanumber"}, config.RabbitMQConfig{})
	if err == nil {
		t.Error("expected error for invalid Redis DB")
	}
}
