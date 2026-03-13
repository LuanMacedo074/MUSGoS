package mus

import (
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
