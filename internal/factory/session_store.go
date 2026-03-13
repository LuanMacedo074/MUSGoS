package factory

import (
	"fmt"
	"strconv"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/config"
	"fsos-server/internal/domain/ports"
)

func NewSessionStore(storeType string, redisCfg config.RedisConfig) (ports.SessionStore, error) {
	switch storeType {
	case "memory":
		return outbound.NewMemorySessionStore(), nil
	case "redis":
		db, err := strconv.Atoi(redisCfg.DB)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB value %q: %w", redisCfg.DB, err)
		}

		connTTL, err := strconv.Atoi(redisCfg.ConnTTL)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_CONN_TTL value %q: %w", redisCfg.ConnTTL, err)
		}

		addr := redisCfg.Host + ":" + redisCfg.Port
		return outbound.NewRedisSessionStore(addr, redisCfg.Password, db, redisCfg.KeyPrefix, connTTL)
	default:
		return nil, fmt.Errorf("unsupported session store type: %s", storeType)
	}
}
