package outbound_test

import (
	"testing"

	"fsos-server/internal/adapters/outbound"
)

func TestRabbitMQQueue_ConnectionFailure(t *testing.T) {
	// RabbitMQ is not expected to be running in test environments.
	// Verify that connection failure returns a meaningful error.
	_, err := outbound.NewRabbitMQQueue("localhost", "15672", "bad", "bad", "/", "test")
	if err == nil {
		t.Error("expected connection error when RabbitMQ is not available")
	}
}
