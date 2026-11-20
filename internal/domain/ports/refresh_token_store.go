package ports

// RefreshTokenStore manages remembered-device credentials independently of their
// persistence backend. Tokens are opaque; only their digests may be persisted.
type RefreshTokenStore interface {
	Issue(userID int64, deviceLabel string) (string, error)
	// Rotate returns zero identity for invalid credentials. Replaying a spent
	// token permanently revokes its device family, even though login is rejected.
	Rotate(token string) (userID int64, rotated string, err error)
	Revoke(token string) error
}
