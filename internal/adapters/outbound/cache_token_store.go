package outbound

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"musgos/internal/domain/ports"
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
	cache             ports.Cache
	ttl               time.Duration
	legacyUserIDClaim string
}

func NewCacheTokenStore(cache ports.Cache, ttl time.Duration) *CacheTokenStore {
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	return &CacheTokenStore{cache: cache, ttl: ttl}
}

// WithLegacyUserIDClaim enables reading flat payloads written before claims were nested.
// The application selects which old claim supplied the transport identity.
func (s *CacheTokenStore) WithLegacyUserIDClaim(key string) *CacheTokenStore {
	s.legacyUserIDClaim = key
	return s
}

type tokenRecord struct {
	UserID string            `json:"user_id"`
	Claims map[string]string `json:"claims"`
}

func (s *CacheTokenStore) Issue(payload ports.TokenPayload) (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	rec, err := json.Marshal(tokenRecord{UserID: payload.UserID, Claims: payload.Claims})
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

	return decodeTokenPayload(data, s.legacyUserIDClaim)
}

func decodeTokenPayload(data []byte, legacyUserIDClaim string) (ports.TokenPayload, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return ports.TokenPayload{}, ports.ErrTokenInvalid
	}
	_, modern := fields["claims"]
	if _, legacy := fields[legacyUserIDClaim]; !modern && legacy && legacyUserIDClaim != "" {
		claims, err := legacyClaims(fields)
		if err != nil {
			return ports.TokenPayload{}, err
		}
		return ports.TokenPayload{UserID: claims[legacyUserIDClaim], Claims: claims}, nil
	}
	payload := ports.TokenPayload{}
	if raw, ok := fields["user_id"]; ok {
		if err := json.Unmarshal(raw, &payload.UserID); err != nil {
			return ports.TokenPayload{}, ports.ErrTokenInvalid
		}
	}
	if raw, ok := fields["claims"]; ok {
		if err := json.Unmarshal(raw, &payload.Claims); err != nil {
			return ports.TokenPayload{}, ports.ErrTokenInvalid
		}
	}
	return payload, nil
}

// legacyClaims preserves string and numeric values from flat token records.
func legacyClaims(fields map[string]json.RawMessage) (map[string]string, error) {
	claims := make(map[string]string, len(fields))
	for key, raw := range fields {
		var value interface{}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, ports.ErrTokenInvalid
		}
		switch value := value.(type) {
		case string:
			claims[key] = value
		case json.Number:
			claims[key] = value.String()
		default:
			return nil, ports.ErrTokenInvalid
		}
	}
	return claims, nil
}
