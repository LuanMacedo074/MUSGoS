package factory

import (
	"fmt"
	"strconv"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/config"
	"fsos-server/internal/domain/ports"
)

func NewMessageQueue(queueType string, queueRedisCfg config.RedisConfig, rabbitCfg config.RabbitMQConfig) (ports.MessageQueue, error) {
	switch queueType {
	case "memory":
		return outbound.NewMemoryQueue(), nil
	case "redis":
		db, err := strconv.Atoi(queueRedisCfg.DB)
		if err != nil {
			return nil, fmt.Errorf("invalid QUEUE_REDIS_DB value %q: %w", queueRedisCfg.DB, err)
		}
		addr := queueRedisCfg.Host + ":" + queueRedisCfg.Port
		return outbound.NewRedisQueue(addr, queueRedisCfg.Password, db, queueRedisCfg.KeyPrefix)
	case "rabbitmq":
		return outbound.NewRabbitMQQueue(
			rabbitCfg.Host, rabbitCfg.Port,
			rabbitCfg.User, rabbitCfg.Password,
			rabbitCfg.VHost, rabbitCfg.Exchange,
		)
	default:
		return nil, fmt.Errorf("unsupported queue type: %s", queueType)
	}
}
