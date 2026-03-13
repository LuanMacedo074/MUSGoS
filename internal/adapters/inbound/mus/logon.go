package mus

import (
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"

	"golang.org/x/crypto/bcrypt"
)

type LogonService struct {
	db               ports.DBAdapter
	sessionStore     ports.SessionStore
	cipher           ports.Cipher
	logger           ports.Logger
	authMode         string
	defaultUserLevel int
}

func NewLogonService(db ports.DBAdapter, sessionStore ports.SessionStore, cipher ports.Cipher, logger ports.Logger, authMode string, defaultUserLevel int) *LogonService {
	return &LogonService{
		db:               db,
		sessionStore:     sessionStore,
		cipher:           cipher,
		logger:           logger,
		authMode:         authMode,
		defaultUserLevel: defaultUserLevel,
	}
}

func (s *LogonService) HandleLogon(clientIP string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	userID, password, err := s.extractCredentials(msg.MsgContent)
	if err != nil {
		if s.authMode == "strict" {
			s.logger.Warn("Logon failed: could not extract credentials", map[string]interface{}{
				"client": clientIP,
				"error":  err.Error(),
			})
			return NewResponse("Logon", "System", []string{msg.SenderID.Value}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
		}
		// In none/open mode, fall back to SenderID
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
				"client": clientIP,
				"userID": userID,
			})
			return NewResponse("Logon", "System", []string{userID}, smus.ErrInvalidUserID, lingo.NewLVoid()), nil
		}

		if errResp := s.validateUserCredentials(user, password, clientIP, userID); errResp != nil {
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
					"client": clientIP,
					"userID": userID,
					"error":  err.Error(),
				})
				return NewResponse("Logon", "System", []string{userID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
			}
		} else {
			if errResp := s.validateUserCredentials(user, password, clientIP, userID); errResp != nil {
				return errResp, nil
			}

			userLevel = user.UserLevel
		}
	}

	// Re-register session with userID instead of the initial clientIP
	s.sessionStore.UnregisterConnection(clientIP)
	s.sessionStore.RegisterConnection(userID, clientIP)

	// Store user level in session for permission checks
	s.sessionStore.SetUserAttribute(userID, "#userLevel", lingo.NewLInteger(int32(userLevel)))

	s.logger.Info("Logon successful", map[string]interface{}{
		"client":     clientIP,
		"userID":     userID,
		"user_level": userLevel,
	})

	return NewResponse("Logon", "System", []string{userID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *LogonService) validateUserCredentials(user *ports.User, password, clientIP, userID string) *smus.MUSMessage {
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Info("Logon failed: invalid password", map[string]interface{}{
			"client": clientIP,
			"userID": userID,
		})
		return NewResponse("Logon", "System", []string{userID}, smus.ErrInvalidPassword, lingo.NewLVoid())
	}

	ban, err := s.db.GetActiveBanByUserID(user.ID)
	if err == nil && ban != nil {
		s.logger.Info("Logon failed: user is banned", map[string]interface{}{
			"client": clientIP,
			"userID": userID,
			"reason": ban.Reason,
		})
		return NewResponse("Logon", "System", []string{userID}, smus.ErrConnectionRefused, lingo.NewLVoid())
	}

	return nil
}

func (s *LogonService) extractCredentials(content lingo.LValue) (userID, password string, err error) {
	if content == nil {
		return "", "", ports.ErrInvalidCredentials
	}

	switch v := content.(type) {
	case *lingo.LList:
		return s.extractFromList(v)
	case *lingo.LPropList:
		return s.extractFromPropList(v)
	default:
		return "", "", ports.ErrInvalidCredentials
	}
}

func (s *LogonService) extractFromList(list *lingo.LList) (string, string, error) {
	if len(list.Values) < 3 {
		return "", "", ports.ErrInvalidCredentials
	}
	// [movieID, userID, password]
	userID := extractStringValue(list.Values[1])
	password := extractStringValue(list.Values[2])
	return userID, password, nil
}

func (s *LogonService) extractFromPropList(plist *lingo.LPropList) (string, string, error) {
	userVal, err := plist.GetElement("userID")
	if err != nil {
		return "", "", err
	}
	passVal, err := plist.GetElement("password")
	if err != nil {
		return "", "", err
	}
	userID := extractStringValue(userVal)
	password := extractStringValue(passVal)
	return userID, password, nil
}

func extractStringValue(v lingo.LValue) string {
	if s, ok := v.(*lingo.LString); ok {
		return s.Value
	}
	return v.String()
}
