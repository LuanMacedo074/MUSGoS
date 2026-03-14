package ports

type RateLimiter interface {
	Allow(key string) bool
	Remove(key string)
}
