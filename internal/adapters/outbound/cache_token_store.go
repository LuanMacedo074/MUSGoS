package outbound

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"fsos-server/internal/domain/ports"
)

// DefaultTokenTTL is the lifetime of an issued auth token. Short by design — a
// token is a single-use bridge between the HTTP login and the immediately
// following MUS Logon, so the replay window is small.
const DefaultTokenTTL = 5 * time.Minute

// tokenKeyPrefix namespaces token keys in the shared cache.
const tokenKeyPrefix = "authtoken:"

// tokenBytes is the entropy of an issued token before hex-encoding.
const tokenBytes = 32

// CacheTokenStore implements ports.TokenStore over the engine's cache. The
// token is an opaque random id; the payload lives server-side under the id, so
// there is nothing in the token to tamper with — an unknown id simply misses.
type CacheTokenStore struct {
	cache ports.Cache
	ttl   time.Duration
}

func NewCacheTokenStore(cache ports.Cache, ttl time.Duration) *CacheTokenStore {
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	return &CacheTokenStore{cache: cache, ttl: ttl}
}

// tokenRecord is the JSON shape stored under a token key. It mirrors
// ports.TokenPayload deliberately: the on-disk serialization format is an
// adapter concern, so the json tags live here rather than leaking into the
// domain port.
type tokenRecord struct {
	CharacterName string `json:"character_name"`
	UserLevel     int    `json:"user_level"`
}

func (s *CacheTokenStore) Issue(payload ports.TokenPayload) (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	rec, err := json.Marshal(tokenRecord{
		CharacterName: payload.CharacterName,
		UserLevel:     payload.UserLevel,
	})
	if err != nil {
		return "", err
	}
	if err := s.cache.Set(tokenKeyPrefix+token, rec, s.ttl); err != nil {
		return "", err
	}
	return token, nil
}

func (s *CacheTokenStore) Validate(token string) (ports.TokenPayload, error) {
	if token == "" {
		return ports.TokenPayload{}, ports.ErrTokenInvalid
	}
	key := tokenKeyPrefix + token

	data, err := s.cache.Get(key)
	if err != nil || data == nil {
		return ports.TokenPayload{}, ports.ErrTokenInvalid
	}

	// Consume before returning: a token authorizes exactly one Logon. Deleting
	// here (even if the payload is corrupt) closes the replay window.
	_ = s.cache.Delete(key)

	var rec tokenRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return ports.TokenPayload{}, ports.ErrTokenInvalid
	}
	return ports.TokenPayload{
		CharacterName: rec.CharacterName,
		UserLevel:     rec.UserLevel,
	}, nil
}
