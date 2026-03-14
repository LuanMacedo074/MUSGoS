package outbound

import (
	"context"
	"fmt"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"

	"github.com/redis/go-redis/v9"
)

type RedisSessionStore struct {
	client    redis.Cmdable
	closer    func() error
	keyPrefix string
	connTTL   time.Duration
}

func NewRedisSessionStore(addr, password string, db int, keyPrefix string, connTTL int) (*RedisSessionStore, error) {
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

	return &RedisSessionStore{
		client:    client,
		closer:    client.Close,
		keyPrefix: keyPrefix,
		connTTL:   time.Duration(connTTL) * time.Second,
	}, nil
}

func NewRedisSessionStoreWithClient(client redis.Cmdable, keyPrefix string, connTTL int) *RedisSessionStore {
	return &RedisSessionStore{
		client:    client,
		closer:    func() error { return nil },
		keyPrefix: keyPrefix,
		connTTL:   time.Duration(connTTL) * time.Second,
	}
}

// --- Key helpers ---

func (r *RedisSessionStore) connKey(clientID string) string {
	return fmt.Sprintf("%s:conn:%s", r.keyPrefix, clientID)
}

func (r *RedisSessionStore) connsKey() string {
	return fmt.Sprintf("%s:conns", r.keyPrefix)
}

func (r *RedisSessionStore) attrKey(clientID string) string {
	return fmt.Sprintf("%s:attr:%s", r.keyPrefix, clientID)
}

func (r *RedisSessionStore) roomKey(roomName string) string {
	return fmt.Sprintf("%s:room:%s", r.keyPrefix, roomName)
}

func (r *RedisSessionStore) clientRoomsKey(clientID string) string {
	return fmt.Sprintf("%s:rooms:%s", r.keyPrefix, clientID)
}

// --- Connection lifecycle ---

func (r *RedisSessionStore) RegisterConnection(clientID, ip string) error {
	ctx := context.Background()
	connKey := r.connKey(clientID)

	pipe := r.client.Pipeline()
	now := time.Now().UTC().Format(time.RFC3339)
	pipe.HSet(ctx, connKey, "ip", ip, "connected_at", now, "last_activity", now)
	if r.connTTL > 0 {
		pipe.Expire(ctx, connKey, r.connTTL)
	}
	pipe.SAdd(ctx, r.connsKey(), clientID)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisSessionStore) UnregisterConnection(clientID string) error {
	ctx := context.Background()

	rooms, err := r.client.SMembers(ctx, r.clientRoomsKey(clientID)).Result()
	if err != nil {
		return fmt.Errorf("failed to get rooms for client %s: %w", clientID, err)
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.connKey(clientID))
	pipe.Del(ctx, r.attrKey(clientID))
	pipe.Del(ctx, r.clientRoomsKey(clientID))
	pipe.SRem(ctx, r.connsKey(), clientID)

	for _, room := range rooms {
		pipe.SRem(ctx, r.roomKey(room), clientID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisSessionStore) GetConnection(clientID string) (*ports.ConnectionInfo, error) {
	ctx := context.Background()
	result, err := r.client.HGetAll(ctx, r.connKey(clientID)).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	connectedAt, _ := time.Parse(time.RFC3339, result["connected_at"])
	lastActivity, _ := time.Parse(time.RFC3339, result["last_activity"])
	return &ports.ConnectionInfo{
		ClientID:       clientID,
		IP:             result["ip"],
		ConnectedAt:    connectedAt,
		LastActivityAt: lastActivity,
	}, nil
}

func (r *RedisSessionStore) GetAllConnections() ([]ports.ConnectionInfo, error) {
	ctx := context.Background()
	clientIDs, err := r.client.SMembers(ctx, r.connsKey()).Result()
	if err != nil {
		return nil, err
	}

	if len(clientIDs) == 0 {
		return []ports.ConnectionInfo{}, nil
	}

	pipe := r.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(clientIDs))
	for i, id := range clientIDs {
		cmds[i] = pipe.HGetAll(ctx, r.connKey(id))
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	conns := make([]ports.ConnectionInfo, 0, len(clientIDs))
	for i, id := range clientIDs {
		result, err := cmds[i].Result()
		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			continue
		}
		connectedAt, _ := time.Parse(time.RFC3339, result["connected_at"])
		lastActivity, _ := time.Parse(time.RFC3339, result["last_activity"])
		conns = append(conns, ports.ConnectionInfo{
			ClientID:       id,
			IP:             result["ip"],
			ConnectedAt:    connectedAt,
			LastActivityAt: lastActivity,
		})
	}
	return conns, nil
}

func (r *RedisSessionStore) UpdateLastActivity(clientID string) error {
	ctx := context.Background()
	return r.client.HSet(ctx, r.connKey(clientID), "last_activity", time.Now().UTC().Format(time.RFC3339)).Err()
}

func (r *RedisSessionStore) IsConnected(clientID string) (bool, error) {
	ctx := context.Background()
	return r.client.SIsMember(ctx, r.connsKey(), clientID).Result()
}

// --- Session attributes ---

func (r *RedisSessionStore) SetUserAttribute(clientID, attrName string, value lingo.LValue) error {
	ctx := context.Background()

	data, err := lingo.MarshalLValue(value)
	if err != nil {
		return err
	}

	return r.client.HSet(ctx, r.attrKey(clientID), attrName, string(data)).Err()
}

func (r *RedisSessionStore) GetUserAttribute(clientID, attrName string) (lingo.LValue, error) {
	ctx := context.Background()

	raw, err := r.client.HGet(ctx, r.attrKey(clientID), attrName).Result()
	if err == redis.Nil {
		return lingo.NewLVoid(), nil
	}
	if err != nil {
		return lingo.NewLVoid(), err
	}

	return lingo.UnmarshalLValue([]byte(raw))
}

func (r *RedisSessionStore) GetUserAttributeNames(clientID string) ([]string, error) {
	ctx := context.Background()

	keys, err := r.client.HKeys(ctx, r.attrKey(clientID)).Result()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *RedisSessionStore) DeleteUserAttribute(clientID, attrName string) error {
	ctx := context.Background()
	return r.client.HDel(ctx, r.attrKey(clientID), attrName).Err()
}

// --- Room management ---

func (r *RedisSessionStore) JoinRoom(roomName, clientID string) error {
	ctx := context.Background()

	pipe := r.client.Pipeline()
	pipe.SAdd(ctx, r.roomKey(roomName), clientID)
	pipe.SAdd(ctx, r.clientRoomsKey(clientID), roomName)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisSessionStore) LeaveRoom(roomName, clientID string) error {
	ctx := context.Background()

	pipe := r.client.Pipeline()
	pipe.SRem(ctx, r.roomKey(roomName), clientID)
	pipe.SRem(ctx, r.clientRoomsKey(clientID), roomName)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisSessionStore) GetRoomMembers(roomName string) ([]string, error) {
	ctx := context.Background()
	return r.client.SMembers(ctx, r.roomKey(roomName)).Result()
}

func (r *RedisSessionStore) GetClientRooms(clientID string) ([]string, error) {
	ctx := context.Background()
	return r.client.SMembers(ctx, r.clientRoomsKey(clientID)).Result()
}

func (r *RedisSessionStore) LeaveAllRooms(clientID string) error {
	ctx := context.Background()

	rooms, err := r.client.SMembers(ctx, r.clientRoomsKey(clientID)).Result()
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()
	for _, room := range rooms {
		pipe.SRem(ctx, r.roomKey(room), clientID)
	}
	pipe.Del(ctx, r.clientRoomsKey(clientID))

	_, err = pipe.Exec(ctx)
	return err
}

// --- Close ---

func (r *RedisSessionStore) Close() error {
	return r.closer()
}

