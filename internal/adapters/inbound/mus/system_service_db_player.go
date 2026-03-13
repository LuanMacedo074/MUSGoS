package mus

import (
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func (s *SystemService) handleDBPlayerGetAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application", "userID", "attribute"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			return s.db.GetPlayerAttribute(lingo.StringValue(f["application"]), lingo.StringValue(f["userID"]), lingo.StringValue(f["attribute"]))
		})
}

func (s *SystemService) handleDBPlayerSetAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application", "userID", "attribute", "value"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			err := s.db.SetPlayerAttribute(lingo.StringValue(f["application"]), lingo.StringValue(f["userID"]), lingo.StringValue(f["attribute"]), f["value"])
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBPlayerDeleteAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application", "userID", "attribute"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			err := s.db.DeletePlayerAttribute(lingo.StringValue(f["application"]), lingo.StringValue(f["userID"]), lingo.StringValue(f["attribute"]))
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBPlayerGetAttributeNames(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application", "userID"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			names, err := s.db.GetPlayerAttributeNames(lingo.StringValue(f["application"]), lingo.StringValue(f["userID"]))
			if err != nil {
				return nil, err
			}
			list := lingo.NewLList()
			for _, name := range names {
				list.Values = append(list.Values, lingo.NewLString(name))
			}
			return list, nil
		})
}
