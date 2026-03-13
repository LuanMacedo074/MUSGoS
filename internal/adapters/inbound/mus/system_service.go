package mus

import (
	"fmt"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"

	"golang.org/x/crypto/bcrypt"
)

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
) *SystemService {
	return &SystemService{
		db:               db,
		sessionStore:     sessionStore,
		cipher:           cipher,
		logger:           logger,
		movieManager:     movieManager,
		groupManager:     groupManager,
		connWriter:       connWriter,
		authMode:         authMode,
		defaultUserLevel: defaultUserLevel,
	}
}

func (s *SystemService) Handle(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	switch msg.Subject.Value {
	case "Logon":
		return s.handleLogon(senderID, msg)
	default:
		s.logger.Warn("Unknown system command", map[string]interface{}{
			"subject":  msg.Subject.Value,
			"senderID": senderID,
		})
		return nil, nil
	}
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
