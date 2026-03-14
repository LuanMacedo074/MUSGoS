package outbound

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client    redis.Cmdable
	closer    func() error
	keyPrefix string
}

func NewRedisCache(addr, password string, db int, keyPrefix string) (*RedisCache, error) {
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

	return &RedisCache{
		client:    client,
		closer:    client.Close,
		keyPrefix: keyPrefix,
	}, nil
}

func NewRedisCacheWithClient(client redis.Cmdable, keyPrefix string) *RedisCache {
	return &RedisCache{
		client:    client,
		closer:    func() error { return nil },
		keyPrefix: keyPrefix,
	}
}

func (c *RedisCache) key(k string) string {
	return fmt.Sprintf("%s:%s", c.keyPrefix, k)
}

func (c *RedisCache) Get(key string) ([]byte, error) {
	ctx := context.Background()
	val, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (c *RedisCache) Set(key string, value []byte, ttl time.Duration) error {
	ctx := context.Background()
	return c.client.Set(ctx, c.key(key), value, ttl).Err()
}

func (c *RedisCache) Delete(key string) error {
	ctx := context.Background()
	return c.client.Del(ctx, c.key(key)).Err()
}

func (c *RedisCache) Exists(key string) (bool, error) {
	ctx := context.Background()
	n, err := c.client.Exists(ctx, c.key(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (c *RedisCache) Close() error {
	return c.closer()
}
