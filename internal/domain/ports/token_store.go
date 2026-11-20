package ports

import "errors"

// ErrTokenInvalid is returned when a token is unknown, already consumed,
// expired, or malformed — every "this token does not grant access" case.
var ErrTokenInvalid = errors.New("token invalid or expired")

// TokenUserLevelClaim is the authorization level interpreted by the Logon service.
const TokenUserLevelClaim = "user_level"

// TokenPayload carries a user identity and application-defined claims across transports.
// The user_level claim supplies the authorization level at Logon.
type TokenPayload struct {
	UserID string
	Claims map[string]string
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
