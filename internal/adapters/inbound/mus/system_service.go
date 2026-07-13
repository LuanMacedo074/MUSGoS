package mus

import (
	"errors"
	"fmt"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

type handlerFunc func(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error)

type SystemService struct {
	db           ports.DBAdapter
	sessionStore ports.SessionStore
	logger       ports.Logger
	movieManager *MovieManager
	groupManager *GroupManager
	connWriter   ports.ConnectionWriter
	logon        *services.LogonService
	authz        *services.Authorizer
	handlers     map[string]handlerFunc
	emailSender  ports.EmailSender
	timerManager ports.TimerManager
}

func NewSystemService(
	db ports.DBAdapter,
	sessionStore ports.SessionStore,
	logger ports.Logger,
	movieManager *MovieManager,
	groupManager *GroupManager,
	connWriter ports.ConnectionWriter,
	logon *services.LogonService,
	authz *services.Authorizer,
	emailSender ports.EmailSender,
	timerManager ports.TimerManager,
) *SystemService {
	s := &SystemService{
		db:           db,
		sessionStore: sessionStore,
		logger:       logger,
		movieManager: movieManager,
		groupManager: groupManager,
		connWriter:   connWriter,
		logon:        logon,
		authz:        authz,
		emailSender:  emailSender,
		timerManager: timerManager,
	}

	s.handlers = map[string]handlerFunc{
		"Logon":                          s.handleLogon,
		"system.server.getVersion":       s.handleServerGetVersion,
		"system.server.getTime":          s.handleServerGetTime,
		"system.server.getUserCount":     s.handleServerGetUserCount,
		"system.server.getMovieCount":    s.handleServerGetMovieCount,
		"system.server.getMovies":        s.handleServerGetMovies,
		"system.movie.getUserCount":      s.handleMovieGetUserCount,
		"system.movie.getGroups":         s.handleMovieGetGroups,
		"system.movie.getGroupCount":     s.handleMovieGetGroupCount,
		"system.group.join":              s.handleGroupJoin,
		"JoinGroup":                      s.handleGroupJoin,
		"system.group.leave":             s.handleGroupLeave,
		"LeaveGroup":                     s.handleGroupLeave,
		"system.group.getUsers":          s.handleGroupGetUsers,
		"system.group.getUserCount":      s.handleGroupGetUserCount,
		"system.group.setAttribute":      s.handleGroupSetAttribute,
		"system.group.getAttribute":      s.handleGroupGetAttribute,
		"system.group.deleteAttribute":   s.handleGroupDeleteAttribute,
		"system.group.getAttributeNames": s.handleGroupGetAttributeNames,
		"system.user.getAddress":         s.handleUserGetAddress,
		"system.user.getGroups":          s.handleUserGetGroups,
		"system.user.delete":             s.handleUserDelete,
		// DBPlayer
		"DBPlayer.getAttribute":      s.handleDBPlayerGetAttribute,
		"DBPlayer.setAttribute":      s.handleDBPlayerSetAttribute,
		"DBPlayer.deleteAttribute":   s.handleDBPlayerDeleteAttribute,
		"DBPlayer.getAttributeNames": s.handleDBPlayerGetAttributeNames,
		// DBApplication
		"DBApplication.getAttribute":      s.handleDBApplicationGetAttribute,
		"DBApplication.setAttribute":      s.handleDBApplicationSetAttribute,
		"DBApplication.deleteAttribute":   s.handleDBApplicationDeleteAttribute,
		"DBApplication.getAttributeNames": s.handleDBApplicationGetAttributeNames,
		// DBAdmin
		"DBAdmin.createApplication": s.handleDBAdminCreateApplication,
		"DBAdmin.deleteApplication": s.handleDBAdminDeleteApplication,
		"DBAdmin.createUser":        s.handleDBAdminCreateUser,
		"DBAdmin.deleteUser":        s.handleDBAdminDeleteUser,
		"DBAdmin.getUserCount":      s.handleDBAdminGetUserCount,
		"DBAdmin.ban":               s.handleDBAdminBan,
		"DBAdmin.revokeBan":         s.handleDBAdminRevokeBan,
		// Email
		"system.server.sendEmail": s.handleServerSendEmail,
		// Kill Timers
		"system.server.setKillTimer":    s.handleServerSetKillTimer,
		"system.server.cancelKillTimer": s.handleServerCancelKillTimer,
		"system.user.setKillTimer":      s.handleUserSetKillTimer,
		"system.user.cancelKillTimer":   s.handleUserCancelKillTimer,
	}

	return s
}

func (s *SystemService) Handle(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	if handler, ok := s.handlers[msg.Subject.Value]; ok {
		return handler(senderID, msg)
	}
	s.logger.Warn("Unknown system command", map[string]interface{}{
		"subject":  msg.Subject.Value,
		"senderID": senderID,
	})
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidServerCommand, lingo.NewLVoid()), nil
}

func (s *SystemService) handleLogon(connectionID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	s.logger.Debug("Logon content", map[string]interface{}{
		"content_type": fmt.Sprintf("%T", msg.MsgContent),
		"content":      msg.MsgContent.String(),
	})

	// Credential *encoding* (positional list vs prop-list) is SMUS protocol;
	// everything after parsing is the LogonService's policy.
	req := services.LogonRequest{ConnectionID: connectionID, SenderID: msg.SenderID.Value}
	if movieID, userID, password, err := s.extractCredentials(msg.MsgContent); err != nil {
		req.ParseErr = err
	} else {
		req.Credentials = &services.LogonCredentials{MovieID: movieID, UserID: userID, Password: password}
	}

	res := s.logon.Logon(req)
	if res.Code != services.LogonOK {
		return NewResponse("Logon", "System", []string{res.UserID}, logonErrCode(res.Code), lingo.NewLVoid()), nil
	}

	// Join movie if movieID was provided and MovieManager is available
	if res.MovieID != "" && s.movieManager != nil {
		if err := s.movieManager.JoinMovie(res.MovieID, res.UserID); err != nil {
			s.logger.Error("Failed to join movie after logon", map[string]interface{}{
				"client":  connectionID,
				"userID":  res.UserID,
				"movieID": res.MovieID,
				"error":   err.Error(),
			})
		} else {
			// Cache movieID for O(1) lookup
			s.sessionStore.SetUserAttribute(res.UserID, "#movieID", lingo.NewLString(res.MovieID))
		}
	}

	s.logger.Info("Logon successful", map[string]interface{}{
		"client":     connectionID,
		"userID":     res.UserID,
		"movieID":    res.MovieID,
		"user_level": res.UserLevel,
	})

	return NewResponse("Logon", "System", []string{res.UserID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

// logonErrCode maps domain logon outcomes to MUS protocol error codes.
func logonErrCode(code services.LogonCode) int32 {
	switch code {
	case services.LogonBadCredentialsFormat:
		return smus.ErrInvalidMessageFormat
	case services.LogonInvalidUser:
		return smus.ErrInvalidUserID
	case services.LogonInvalidPassword:
		return smus.ErrInvalidPassword
	case services.LogonRefused:
		return smus.ErrConnectionRefused
	default:
		return smus.ErrServerInternalError
	}
}

func (s *SystemService) extractCredentials(content lingo.LValue) (movieID, userID, password string, err error) {
	if content == nil {
		return "", "", "", ports.ErrInvalidCredentials
	}

	switch v := content.(type) {
	case *lingo.LList:
		return s.extractFromList(v)
	case *lingo.LPropList:
		return s.extractFromPropList(v)
	default:
		return "", "", "", ports.ErrInvalidCredentials
	}
}

func (s *SystemService) extractFromList(list *lingo.LList) (string, string, string, error) {
	if len(list.Values) < 3 {
		return "", "", "", ports.ErrInvalidCredentials
	}
	movieID := lingo.StringValue(list.Values[0])
	userID := lingo.StringValue(list.Values[1])
	password := lingo.StringValue(list.Values[2])
	return movieID, userID, password, nil
}

func (s *SystemService) getUserMovieID(userID string) (string, error) {
	val, err := s.sessionStore.GetUserAttribute(userID, "#movieID")
	if err != nil {
		return "", fmt.Errorf("user %q is not in any movie", userID)
	}
	if str, ok := val.(*lingo.LString); ok && str.Value != "" {
		return str.Value, nil
	}
	return "", fmt.Errorf("user %q is not in any movie", userID)
}

// errCrossUserDenied is returned by a DB action when the caller tries to touch
// another user's data without an admin-level session; dbErrorCode maps it to a
// command-refused response. (backlog H2)
var errCrossUserDenied = errors.New("cross-user access denied")

// handleDBCommand is a generic helper for DB command handlers that follow the pattern:
// check permissions → parse proplist → extract required fields → execute action → return response.
func (s *SystemService) handleDBCommand(senderID string, msg *smus.MUSMessage,
	requiredFields []string,
	action func(fields map[string]lingo.LValue) (lingo.LValue, error),
) (*smus.MUSMessage, error) {
	if !s.authz.CanRun(senderID, msg.Subject.Value) {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidServerCommand, lingo.NewLVoid()), nil
	}
	fields := make(map[string]lingo.LValue, len(requiredFields))
	if len(requiredFields) > 0 {
		plist, ok := msg.MsgContent.(*lingo.LPropList)
		if !ok {
			return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
		}
		for _, name := range requiredFields {
			val, err := plist.GetElement(name)
			if err != nil {
				return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
			}
			fields[name] = val
		}
	}
	result, err := action(fields)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, dbErrorCode(err), lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, result), nil
}

// dbErrorCode maps domain errors to MUS protocol error codes.
func dbErrorCode(err error) int32 {
	switch {
	case errors.Is(err, errCrossUserDenied):
		return smus.ErrInvalidServerCommand
	case errors.Is(err, ports.ErrUserNotFound):
		return smus.ErrDatabaseUserIDNotFound
	case errors.Is(err, ports.ErrBanNotFound):
		return smus.ErrDatabaseDataNotFound
	default:
		return smus.ErrServerInternalError
	}
}

func (s *SystemService) extractFromPropList(plist *lingo.LPropList) (string, string, string, error) {
	userVal, err := plist.GetElement("userID")
	if err != nil {
		return "", "", "", err
	}
	passVal, err := plist.GetElement("password")
	if err != nil {
		return "", "", "", err
	}
	userID := lingo.StringValue(userVal)
	password := lingo.StringValue(passVal)

	var movieID string
	if movieVal, err := plist.GetElement("movieID"); err == nil {
		movieID = lingo.StringValue(movieVal)
	}

	return movieID, userID, password, nil
}
