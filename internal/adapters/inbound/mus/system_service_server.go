package mus

import (
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
	"time"
)

const ServerVersion = "MUSGoS/0.1.0"

func (s *SystemService) handleServerGetVersion(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLString(ServerVersion)), nil
}

func (s *SystemService) handleServerGetTime(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	// NOTE: The MUS protocol uses int32 for timestamps (Lingo's LInteger is int32 by design).
	// This will overflow in 2038 — a known limitation of the original protocol format.
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLInteger(int32(time.Now().Unix()))), nil
}

func (s *SystemService) handleServerGetUserCount(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	conns, err := s.sessionStore.GetAllConnections()
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLInteger(int32(len(conns)))), nil
}

func (s *SystemService) handleServerGetMovieCount(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	movies := s.movieManager.GetMovies()
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLInteger(int32(len(movies)))), nil
}

func (s *SystemService) handleServerGetMovies(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	movies := s.movieManager.GetMovies()
	list := lingo.NewLList()
	for _, m := range movies {
		list.Values = append(list.Values, lingo.NewLString(m))
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, list), nil
}

func (s *SystemService) handleServerSendEmail(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	if !s.checkCommandLevel(senderID, msg.Subject.Value) {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidServerCommand, lingo.NewLVoid()), nil
	}

	if s.emailSender == nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}

	plist, ok := msg.MsgContent.(*lingo.LPropList)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}

	emailMsg := &ports.EmailMessage{}
	if v, err := plist.GetElement("sender"); err == nil {
		emailMsg.From = lingo.StringValue(v)
	}
	if v, err := plist.GetElement("recipient"); err == nil {
		emailMsg.To = lingo.StringValue(v)
	}
	if v, err := plist.GetElement("subject"); err == nil {
		emailMsg.Subject = lingo.StringValue(v)
	}
	if v, err := plist.GetElement("SMTPhost"); err == nil {
		emailMsg.SMTPHost = lingo.StringValue(v)
	}
	if v, err := plist.GetElement("data"); err == nil {
		emailMsg.Body = lingo.StringValue(v)
	}

	if emailMsg.To == "" || emailMsg.SMTPHost == "" {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}

	if err := s.emailSender.SendEmail(emailMsg); err != nil {
		s.logger.Error("Failed to send email", map[string]interface{}{
			"error": err.Error(),
		})
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}

	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *SystemService) handleServerSetKillTimer(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	if !s.checkCommandLevel(senderID, msg.Subject.Value) {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidServerCommand, lingo.NewLVoid()), nil
	}

	if s.timerManager == nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}

	minutes := int(msg.MsgContent.ToInteger())
	if minutes <= 0 {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	s.timerManager.SetServerKillTimer(minutes)

	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *SystemService) handleServerCancelKillTimer(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	if !s.checkCommandLevel(senderID, msg.Subject.Value) {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidServerCommand, lingo.NewLVoid()), nil
	}

	if s.timerManager == nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}

	s.timerManager.CancelServerKillTimer()

	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}
