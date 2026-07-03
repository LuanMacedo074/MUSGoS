//go:build integration

// Integration tests run against real Postgres/Redis/RabbitMQ servers brought up
// by docker/thirdparties. They compile only under `-tags=integration` and each
// has "Integration" in its name so the runner can target them with -run.
//
// Connection details come from TEST_* env vars (set by scripts/run-tests.sh from
// docker/thirdparties/custom_settings.env); the defaults below match that file,
// so `go test -tags=integration ./...` works out of the box once the services
// are up. If a service is unreachable the test skips with a hint rather than
// failing, so a tagged run on a machine without the services degrades cleanly.
package outbound_test

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
)

// runNonce namespaces every Redis key to this test run, so re-running the suite
// against a persistent Redis (a container not recreated between runs) never
// trips over keys left by a previous run. CI uses a fresh container each time;
// this just makes local reruns robust too.
var runNonce = strconv.FormatInt(time.Now().UnixNano(), 36)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func pgDSN() string {
	return envOr("TEST_POSTGRES_DSN", "postgres://postgres:my_secret_pw@127.0.0.1:5432/musgo_regression?sslmode=disable")
}

func redisAddr() string     { return envOr("TEST_REDIS_ADDR", "127.0.0.1:6379") }
func redisPassword() string { return os.Getenv("TEST_REDIS_PASSWORD") }
func redisDB() int          { n, _ := strconv.Atoi(envOr("TEST_REDIS_DB", "0")); return n }

// safeName turns a test name into a Redis-key-safe, unique-per-test prefix so
// tests sharing one server don't collide.
func safeName(t *testing.T) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
}

func newRedisCache(t *testing.T) *outbound.RedisCache {
	t.Helper()
	c, err := outbound.NewRedisCache(redisAddr(), redisPassword(), redisDB(), "musgotest:"+runNonce+":cache:"+safeName(t))
	if err != nil {
		t.Skipf("redis not reachable at %s (run 'make thirdparties-up'): %v", redisAddr(), err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func newRedisQueue(t *testing.T) *outbound.RedisQueue {
	t.Helper()
	q, err := outbound.NewRedisQueue(redisAddr(), redisPassword(), redisDB(), "musgotest:"+runNonce+":queue:"+safeName(t))
	if err != nil {
		t.Skipf("redis not reachable at %s (run 'make thirdparties-up'): %v", redisAddr(), err)
	}
	t.Cleanup(func() { _ = q.Close() })
	return q
}

func newRedisSessionStore(t *testing.T) *outbound.RedisSessionStore {
	t.Helper()
	s, err := outbound.NewRedisSessionStore(redisAddr(), redisPassword(), redisDB(), "musgotest:"+runNonce+":sess:"+safeName(t), 3600)
	if err != nil {
		t.Skipf("redis not reachable at %s (run 'make thirdparties-up'): %v", redisAddr(), err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func newRabbitMQ(t *testing.T) *outbound.RabbitMQQueue {
	t.Helper()
	q, err := outbound.NewRabbitMQQueue(
		envOr("TEST_RABBITMQ_HOST", "127.0.0.1"),
		envOr("TEST_RABBITMQ_PORT", "5672"),
		envOr("TEST_RABBITMQ_USER", "guest"),
		envOr("TEST_RABBITMQ_PASSWORD", "guest"),
		envOr("TEST_RABBITMQ_VHOST", "/"),
		"musgo_test_exchange",
	)
	if err != nil {
		t.Skipf("rabbitmq not reachable (run 'make thirdparties-up'): %v", err)
	}
	t.Cleanup(func() { _ = q.Close() })
	return q
}
