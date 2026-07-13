package services

import (
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

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

// LogonService owns the logon use case: auth-mode policy, credential
// validation, ban rejection, the session-takeover guard, connection
// remapping, and session (re-)registration under the authenticated identity.
type LogonService struct {
	db           ports.DBAdapter
	sessions     ports.SessionStore
	connWriter   ports.ConnectionWriter
	logger       ports.Logger
	mode         string // "none", "open" (default), or "strict"
	defaultLevel int
}

func NewLogonService(
	db ports.DBAdapter,
	sessions ports.SessionStore,
	connWriter ports.ConnectionWriter,
	logger ports.Logger,
	mode string,
	defaultLevel int,
) *LogonService {
	return &LogonService{
		db:           db,
		sessions:     sessions,
		connWriter:   connWriter,
		logger:       logger,
		mode:         mode,
		defaultLevel: defaultLevel,
	}
}

// Logon runs the full logon use case and reports the outcome. On LogonOK the
// session is registered under the effective userID with the user level
// stamped; on any other code no session state has been taken over.
func (s *LogonService) Logon(req LogonRequest) LogonResult {
	userID := req.Credentials.UserID
	movieID := req.Credentials.MovieID

	userLevel := s.defaultLevel

	return s.establishSession(req.ConnectionID, userID, movieID, userLevel)
}

// establishSession is the mode-independent tail of a successful
// authentication: takeover guard, connection remap, re-registration under
// userID (preserving the client's real IP), and user-level stamping.
func (s *LogonService) establishSession(connectionID, userID, movieID string, userLevel int) LogonResult {
	// Reject a Logon for a userID that already has a live session: otherwise a
	// second client could remap the connection and hijack/evict the first. (H3)
	if userID != connectionID {
		if existing, _ := s.sessions.GetConnection(userID); existing != nil {
			s.logger.Warn("Logon rejected: userID already connected", map[string]interface{}{
				"client": connectionID,
				"userID": userID,
			})
			return LogonResult{Code: LogonRefused, UserID: userID}
		}
	}

	// Remap the connection so future messages use userID. Do this before touching
	// the session store: if the slot is already bound to another connection (race
	// with a concurrent logon) RemapClientID returns false and we refuse rather
	// than clobber it. (H3)
	if s.connWriter != nil {
		if !s.connWriter.RemapClientID(connectionID, userID) {
			s.logger.Warn("Logon rejected: userID already bound to another connection", map[string]interface{}{
				"client": connectionID,
				"userID": userID,
			})
			return LogonResult{Code: LogonRefused, UserID: userID}
		}
	}

	// Re-register the session under userID, preserving the client's real IP from the
	// initial registration instead of storing the connection id in the IP field (L5).
	ip := connectionID
	if existing, _ := s.sessions.GetConnection(connectionID); existing != nil && existing.IP != "" {
		ip = existing.IP
	}
	s.sessions.UnregisterConnection(connectionID)
	s.sessions.RegisterConnection(userID, ip)

	// Store user level in session for permission checks
	s.sessions.SetUserAttribute(userID, UserLevelAttribute, lingo.NewLInteger(int32(userLevel)))

	return LogonResult{Code: LogonOK, UserID: userID, UserLevel: userLevel, MovieID: movieID}
}
