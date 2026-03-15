package ports

import "time"

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
	SetAdd(key, member string) error
	SetRemove(key, member string) error
	SetMembers(key string) ([]string, error)
	SetIsMember(key, member string) (bool, error)
	Close() error
}
