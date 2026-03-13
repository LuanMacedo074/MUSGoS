package mus

import (
	"errors"
	"fmt"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"

	"golang.org/x/crypto/bcrypt"
)

type handlerFunc func(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error)

type SystemService struct {
	db               ports.DBAdapter
	sessionStore     ports.SessionStore
	cipher           ports.Cipher
	logger           ports.Logger
	movieManager     *MovieManager
	groupManager     *GroupManager
	connWriter       ports.ConnectionWriter
	authMode         string
	defaultUserLevel int
	commandLevels    map[string]int
	handlers         map[string]handlerFunc
}

func NewSystemService(
	db ports.DBAdapter,
	sessionStore ports.SessionStore,
	cipher ports.Cipher,
	logger ports.Logger,
	movieManager *MovieManager,
	groupManager *GroupManager,
	connWriter ports.ConnectionWriter,
	authMode string,
	defaultUserLevel int,
	commandLevels map[string]int,
) *SystemService {
	s := &SystemService{
		db:               db,
		sessionStore:     sessionStore,
		cipher:           cipher,
		logger:           logger,
		movieManager:     movieManager,
		groupManager:     groupManager,
		connWriter:       connWriter,
		authMode:         authMode,
		defaultUserLevel: defaultUserLevel,
		commandLevels:    commandLevels,
	}

	s.handlers = map[string]handlerFunc{
		"Logon":                        s.handleLogon,
		"system.server.getVersion":     s.handleServerGetVersion,
		"system.server.getTime":        s.handleServerGetTime,
		"system.server.getUserCount":   s.handleServerGetUserCount,
		"system.server.getMovieCount":  s.handleServerGetMovieCount,
		"system.server.getMovies":      s.handleServerGetMovies,
		"system.movie.getUserCount":    s.handleMovieGetUserCount,
		"system.movie.getGroups":       s.handleMovieGetGroups,
		"system.movie.getGroupCount":   s.handleMovieGetGroupCount,
		"system.group.join":            s.handleGroupJoin,
		"system.group.leave":           s.handleGroupLeave,
		"system.group.getUsers":        s.handleGroupGetUsers,
		"system.group.getUserCount":    s.handleGroupGetUserCount,
		"system.group.setAttribute":    s.handleGroupSetAttribute,
		"system.group.getAttribute":    s.handleGroupGetAttribute,
		"system.group.deleteAttribute": s.handleGroupDeleteAttribute,
		"system.group.getAttributeNames": s.handleGroupGetAttributeNames,
		"system.user.getAddress":       s.handleUserGetAddress,
		"system.user.getGroups":        s.handleUserGetGroups,
		"system.user.delete":           s.handleUserDelete,
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
	movieID, userID, password, err := s.extractCredentials(msg.MsgContent)
	if err != nil {
		if s.authMode == "strict" {
			s.logger.Warn("Logon failed: could not extract credentials", map[string]interface{}{
				"client": connectionID,
				"error":  err.Error(),
			})
			return NewResponse("Logon", "System", []string{msg.SenderID.Value}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
		}
		userID = msg.SenderID.Value
		password = ""
	}

	userLevel := s.defaultUserLevel

	switch s.authMode {
	case "none":
		// Accept any user without DB lookup

	case "strict":
		user, err := s.db.GetUser(userID)
		if err != nil {
			s.logger.Info("Logon failed: user not found", map[string]interface{}{
				"client": connectionID,
				"userID": userID,
			})
			return NewResponse("Logon", "System", []string{userID}, smus.ErrInvalidUserID, lingo.NewLVoid()), nil
		}

		if errResp := s.validateUserCredentials(user, password, connectionID, userID); errResp != nil {
			return errResp, nil
		}

		userLevel = user.UserLevel

	default: // "open"
		user, err := s.db.GetUser(userID)
		if err != nil {
			if err == ports.ErrUserNotFound {
				// No DB record — accept anyway in open mode, use defaultUserLevel
			} else {
				s.logger.Error("Logon failed: database error", map[string]interface{}{
					"client": connectionID,
					"userID": userID,
					"error":  err.Error(),
				})
				return NewResponse("Logon", "System", []string{userID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
			}
		} else {
			if errResp := s.validateUserCredentials(user, password, connectionID, userID); errResp != nil {
				return errResp, nil
			}

			userLevel = user.UserLevel
		}
	}

	// Re-register session with userID instead of the initial connectionID
	s.sessionStore.UnregisterConnection(connectionID)
	s.sessionStore.RegisterConnection(userID, connectionID)

	// Remap the connection so future messages use userID
	if s.connWriter != nil {
		s.connWriter.RemapClientID(connectionID, userID)
	}

	// Store user level in session for permission checks
	s.sessionStore.SetUserAttribute(userID, "#userLevel", lingo.NewLInteger(int32(userLevel)))

	// Join movie if movieID was provided and MovieManager is available
	if movieID != "" && s.movieManager != nil {
		if err := s.movieManager.JoinMovie(movieID, userID); err != nil {
			s.logger.Error("Failed to join movie after logon", map[string]interface{}{
				"client":  connectionID,
				"userID":  userID,
				"movieID": movieID,
				"error":   err.Error(),
			})
		} else {
			// Cache movieID for O(1) lookup
			s.sessionStore.SetUserAttribute(userID, "#movieID", lingo.NewLString(movieID))
		}
	}

	s.logger.Info("Logon successful", map[string]interface{}{
		"client":     connectionID,
		"userID":     userID,
		"movieID":    movieID,
		"user_level": userLevel,
	})

	return NewResponse("Logon", "System", []string{userID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *SystemService) validateUserCredentials(user *ports.User, password, connectionID, userID string) *smus.MUSMessage {
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Info("Logon failed: invalid password", map[string]interface{}{
			"client": connectionID,
			"userID": userID,
		})
		return NewResponse("Logon", "System", []string{userID}, smus.ErrInvalidPassword, lingo.NewLVoid())
	}

	ban, err := s.db.GetActiveBanByUserID(user.ID)
	if err == nil && ban != nil {
		s.logger.Info("Logon failed: user is banned", map[string]interface{}{
			"client": connectionID,
			"userID": userID,
			"reason": ban.Reason,
		})
		return NewResponse("Logon", "System", []string{userID}, smus.ErrConnectionRefused, lingo.NewLVoid())
	}

	return nil
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

func (s *SystemService) getUserLevel(userID string) int {
	val, err := s.sessionStore.GetUserAttribute(userID, "#userLevel")
	if err != nil {
		return 0
	}
	return int(val.ToInteger())
}

func (s *SystemService) checkCommandLevel(senderID, command string) bool {
	requiredLevel, ok := s.commandLevels[command]
	if !ok {
		return false
	}
	return s.getUserLevel(senderID) >= requiredLevel
}

// handleDBCommand is a generic helper for DB command handlers that follow the pattern:
// check permissions → parse proplist → extract required fields → execute action → return response.
func (s *SystemService) handleDBCommand(senderID string, msg *smus.MUSMessage,
	requiredFields []string,
	action func(fields map[string]lingo.LValue) (lingo.LValue, error),
) (*smus.MUSMessage, error) {
	if !s.checkCommandLevel(senderID, msg.Subject.Value) {
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
