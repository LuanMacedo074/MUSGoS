package mus

import (
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func (s *SystemService) handleDBApplicationGetAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application", "attribute"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			return s.db.GetApplicationAttribute(lingo.StringValue(f["application"]), lingo.StringValue(f["attribute"]))
		})
}

func (s *SystemService) handleDBApplicationSetAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application", "attribute", "value"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			err := s.db.SetApplicationAttribute(lingo.StringValue(f["application"]), lingo.StringValue(f["attribute"]), f["value"])
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBApplicationDeleteAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application", "attribute"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			err := s.db.DeleteApplicationAttribute(lingo.StringValue(f["application"]), lingo.StringValue(f["attribute"]))
			return lingo.NewLVoid(), err
		})
}

func (s *SystemService) handleDBApplicationGetAttributeNames(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return s.handleDBCommand(senderID, msg, []string{"application"},
		func(f map[string]lingo.LValue) (lingo.LValue, error) {
			names, err := s.db.GetApplicationAttributeNames(lingo.StringValue(f["application"]))
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
