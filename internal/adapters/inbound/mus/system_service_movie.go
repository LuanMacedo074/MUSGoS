package mus

import (
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func (s *SystemService) handleMovieGetUserCount(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	count, err := s.movieManager.GetMovieUserCount(movieID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLInteger(int32(count))), nil
}

func (s *SystemService) handleMovieGetGroups(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	movie, ok := s.movieManager.GetMovie(movieID)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	names := movie.GetGroupNames()
	list := lingo.NewLList()
	for _, name := range names {
		list.Values = append(list.Values, lingo.NewLString(name))
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, list), nil
}

func (s *SystemService) handleMovieGetGroupCount(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	movie, ok := s.movieManager.GetMovie(movieID)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLInteger(int32(movie.GetGroupCount()))), nil
}
