package services

// UserLevelAttribute is the session attribute the logon flow stamps with the
// authenticated user's level and the authorization checks read back. It is
// owned here so the write and read sides can never drift.
const UserLevelAttribute = "#userLevel"

// LogonCode classifies a logon outcome in domain vocabulary. Only the
// protocol adapter maps codes to MUS wire error codes.
type LogonCode int

const (
	LogonOK LogonCode = iota
	// LogonBadCredentialsFormat: credentials were required (strict mode) but
	// could not be parsed from the message.
	LogonBadCredentialsFormat
	// LogonInvalidUser: the userID has no account (strict mode only).
	LogonInvalidUser
	// LogonInvalidPassword: the account exists but the password check failed.
	LogonInvalidPassword
	// LogonRefused: the connection is not allowed — banned user, or the
	// userID already has a live session (takeover guard).
	LogonRefused
	// LogonInternalError: an infrastructure failure interrupted the attempt.
	LogonInternalError
)

// LogonCredentials are the parsed logon credentials. How they were encoded on
// the wire (positional list vs prop-list) is the adapter's concern.
type LogonCredentials struct {
	MovieID  string
	UserID   string
	Password string
}

// LogonRequest is a protocol-neutral logon attempt. Credentials is nil when
// the adapter could not parse credentials from the message; ParseErr then
// carries the reason (for logging) and the mode policy decides whether to
// fall back to the wire sender identity or refuse.
type LogonRequest struct {
	ConnectionID string
	SenderID     string
	Credentials  *LogonCredentials
	ParseErr     error
}

// LogonResult is the outcome of a logon attempt. UserID is the effective
// identity the attempt was resolved against (also the response recipient);
// MovieID is echoed from the credentials for the adapter's post-logon movie
// join.
type LogonResult struct {
	Code      LogonCode
	UserID    string
	UserLevel int
	MovieID   string
}
