package factory

import (
	"fmt"
	"strconv"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/config"
	"fsos-server/internal/domain/ports"
)

func NewCache(cacheType string, redisCfg config.RedisConfig) (ports.Cache, error) {
	switch cacheType {
	case "memory":
		return outbound.NewMemoryCache(), nil
	case "redis":
		db, err := strconv.Atoi(redisCfg.DB)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_REDIS_DB value %q: %w", redisCfg.DB, err)
		}
		addr := redisCfg.Host + ":" + redisCfg.Port
		return outbound.NewRedisCache(addr, redisCfg.Password, db, redisCfg.KeyPrefix)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
	}
}
