package outbound_test

import (
	"context"
	"testing"

	"fsos-server/internal/adapters/outbound"

	"github.com/redis/go-redis/v9"
)

const testKeyPrefix = "musgo_test"

func skipIfNoRedis(t *testing.T) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
}

func newRedisTestStore(t *testing.T) *outbound.RedisSessionStore {
	t.Helper()
	skipIfNoRedis(t)

	store, err := outbound.NewRedisSessionStore("localhost:6379", "", 0, testKeyPrefix, 3600)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	t.Cleanup(func() {
		client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		defer client.Close()
		iter := client.Scan(context.Background(), 0, testKeyPrefix+":*", 0).Iterator()
		for iter.Next(context.Background()) {
			client.Del(context.Background(), iter.Val())
		}
		store.Close()
	})

	return store
}

// --- Connection lifecycle ---

func TestRedis_RegisterConnection(t *testing.T) {
	testRegisterConnection(t, newRedisTestStore(t))
}

func TestRedis_IsConnected(t *testing.T) {
	testIsConnected(t, newRedisTestStore(t))
}

func TestRedis_GetAllConnections(t *testing.T) {
	testGetAllConnections(t, newRedisTestStore(t))
}

func TestRedis_UnregisterConnection_CleansEverything(t *testing.T) {
	testUnregisterConnection_CleansEverything(t, newRedisTestStore(t))
}

// --- Session attributes ---

func TestRedis_SessionAttribute_SetGetDelete(t *testing.T) {
	testSessionAttribute_SetGetDelete(t, newRedisTestStore(t))
}

func TestRedis_SessionAttribute_Float(t *testing.T) {
	testSessionAttribute_Float(t, newRedisTestStore(t))
}

func TestRedis_SessionAttribute_String(t *testing.T) {
	testSessionAttribute_String(t, newRedisTestStore(t))
}

func TestRedis_SessionAttribute_Symbol(t *testing.T) {
	testSessionAttribute_Symbol(t, newRedisTestStore(t))
}

// --- Room management ---

func TestRedis_JoinRoom_And_GetMembers(t *testing.T) {
	testJoinRoom_And_GetMembers(t, newRedisTestStore(t))
}

func TestRedis_LeaveRoom(t *testing.T) {
	testLeaveRoom(t, newRedisTestStore(t))
}

func TestRedis_GetClientRooms(t *testing.T) {
	testGetClientRooms(t, newRedisTestStore(t))
}

func TestRedis_LeaveAllRooms(t *testing.T) {
	testLeaveAllRooms(t, newRedisTestStore(t))
}

func TestRedis_GetConnection_NonExistent(t *testing.T) {
	testGetConnection_NonExistent(t, newRedisTestStore(t))
}
