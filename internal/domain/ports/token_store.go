package ports

import "errors"

// ErrTokenInvalid is returned when a token is unknown, already consumed,
// expired, or malformed — every "this token does not grant access" case.
var ErrTokenInvalid = errors.New("token invalid or expired")

// TokenPayload is the identity a token carries across the HTTP→MUS socket gap:
// the character the session will bind to and the level to stamp on it. The
// engine trusts this payload — it never re-reads any account table.
type TokenPayload struct {
	CharacterName string
	UserLevel     int
}

// TokenStore issues short-TTL, single-use tokens carrying a TokenPayload and
// validates them. Validate consumes the token, so a token authorizes exactly
// one Logon. Backed by the cache the engine already runs (Redis in prod).
type TokenStore interface {
	// Issue mints an opaque token bound to payload with the store's TTL.
	Issue(payload TokenPayload) (string, error)
	// Validate looks up and consumes the token, returning its payload; it
	// returns ErrTokenInvalid for any unknown/expired/consumed/malformed token.
	Validate(token string) (TokenPayload, error)
}
