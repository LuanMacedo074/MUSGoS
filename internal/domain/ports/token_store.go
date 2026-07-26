package ports

import "errors"

// ErrTokenInvalid is returned when a token is unknown, already consumed,
// expired, or malformed — every "this token does not grant access" case.
var ErrTokenInvalid = errors.New("token invalid or expired")

// TokenPayload is the identity a token carries across the HTTP→MUS socket gap:
// the character the session will bind to and the level to stamp on it. The
// engine trusts this payload — it never re-reads any account table.
//
// AccountID carries the authenticated account through the pre-Logon character
// flow: a character-less account logs in (CharacterName empty) and its create
// call must attach the new character to the account it just proved it owns,
// without re-authenticating or trusting a client-supplied id. The Logon token
// path ignores AccountID (it binds on CharacterName + UserLevel only).
type TokenPayload struct {
	CharacterName string
	UserLevel     int
	AccountID     int
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
