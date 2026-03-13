package mus

import (
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
	"strings"
)

func (s *SystemService) handleUserGetAddress(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	conn, err := s.sessionStore.GetConnection(senderID)
	if err != nil || conn == nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLString(conn.IP)), nil
}

func (s *SystemService) handleUserGetGroups(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}

	rooms, err := s.sessionStore.GetClientRooms(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}

	prefix := movieID + ":"
	movieRoom := "movie:" + movieID
	list := lingo.NewLList()
	for _, room := range rooms {
		if room == movieRoom {
			continue
		}
		if strings.HasPrefix(room, prefix) {
			groupName := strings.TrimPrefix(room, prefix)
			list.Values = append(list.Values, lingo.NewLString(groupName))
		}
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, list), nil
}

func (s *SystemService) handleUserDelete(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	if !s.checkCommandLevel(senderID, msg.Subject.Value) {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidServerCommand, lingo.NewLVoid()), nil
	}

	targetUserID, err := lingo.ExtractString(msg.MsgContent)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}

	// Clean up session state before disconnecting to avoid ghost users in rooms
	s.sessionStore.LeaveAllRooms(targetUserID)
	s.sessionStore.UnregisterConnection(targetUserID)

	if s.connWriter != nil {
		s.connWriter.DisconnectClient(targetUserID)
	}

	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}
