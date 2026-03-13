package mus

import (
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"

	"golang.org/x/crypto/bcrypt"
)

func (s *SystemService) handleDBAdminCreateApplication(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			err := s.db.CreateApplication(lingo.StringValue(f["application"]))
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBAdminDeleteApplication(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			err := s.db.DeleteApplication(lingo.StringValue(f["application"]))
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBAdminCreateUser(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"userID", "password", "userLevel"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			hash, err := bcrypt.GenerateFromPassword([]byte(lingo.StringValue(f["password"])), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			err = s.db.CreateUser(lingo.StringValue(f["userID"]), string(hash), int(f["userLevel"].ToInteger()))
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBAdminDeleteUser(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"userID"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			err := s.db.DeleteUser(lingo.StringValue(f["userID"]))
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBAdminGetUserCount(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, nil,
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			conns, err := s.sessionStore.GetAllConnections()
			if err != nil {
				return nil, err
			}
			return lingo.NewLInteger(int32(len(conns))), nil
		})
}

func (s *SystemService) handleDBAdminBan(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"userID", "reason"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			user, err := s.db.GetUser(lingo.StringValue(f["userID"]))
			if err != nil {
				return nil, err
			}
			err = s.db.CreateBan(&user.ID, nil, lingo.StringValue(f["reason"]), nil)
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBAdminRevokeBan(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"userID"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			user, err := s.db.GetUser(lingo.StringValue(f["userID"]))
			if err != nil {
				return nil, err
			}
			ban, err := s.db.GetActiveBanByUserID(user.ID)
			if err != nil {
				return nil, err
			}
			err = s.db.RevokeBan(ban.ID)
			return lingo.NewLVoid(), err
		})
}
