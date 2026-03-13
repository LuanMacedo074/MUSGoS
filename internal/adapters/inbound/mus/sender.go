package mus

import (
	"fmt"
	"strings"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

type Sender struct {
	connWriter   ports.ConnectionWriter
	sessionStore ports.SessionStore
	logger       ports.Logger
	cipher       ports.Cipher
	allEncrypted bool
}

func NewSender(connWriter ports.ConnectionWriter, sessionStore ports.SessionStore, logger ports.Logger, cipher ports.Cipher, allEncrypted bool) *Sender {
	return &Sender{
		connWriter:   connWriter,
		sessionStore: sessionStore,
		logger:       logger,
		cipher:       cipher,
		allEncrypted: allEncrypted,
	}
}

func (s *Sender) SendMessage(senderID, recipientID, subject string, content lingo.LValue) error {
	if strings.HasPrefix(recipientID, "@") {
		return s.deliverToGroup(senderID, recipientID, subject, content)
	}

	msg := NewResponse(subject, senderID, []string{recipientID}, smus.ErrNoError, content)
	msgBytes := msg.GetBytes()
	if s.allEncrypted && s.cipher != nil {
		msgBytes = s.cipher.Encrypt(msgBytes)
	}
	return s.connWriter.WriteToClient(recipientID, msgBytes)
}

func (s *Sender) deliverToGroup(senderID, groupRef, subject string, content lingo.LValue) error {
	// Find the movie the sender is in
	rooms, err := s.sessionStore.GetClientRooms(senderID)
	if err != nil {
		return fmt.Errorf("failed to get sender rooms: %w", err)
	}

	var movieID string
	for _, room := range rooms {
		if strings.HasPrefix(room, "movie:") {
			movieID = strings.TrimPrefix(room, "movie:")
			break
		}
	}
	if movieID == "" {
		return fmt.Errorf("sender %q is not in any movie", senderID)
	}

	// Look up group members via session store room
	roomName := groupRoomName(movieID, groupRef)
	members, err := s.sessionStore.GetRoomMembers(roomName)
	if err != nil {
		return fmt.Errorf("failed to get group members for %s: %w", groupRef, err)
	}

	// Serialize once with the group reference as recipient, then deliver to all members
	msg := NewResponse(subject, senderID, []string{groupRef}, smus.ErrNoError, content)
	msgBytes := msg.GetBytes()
	if s.allEncrypted && s.cipher != nil {
		msgBytes = s.cipher.Encrypt(msgBytes)
	}

	for _, memberID := range members {
		if err := s.connWriter.WriteToClient(memberID, msgBytes); err != nil {
			s.logger.Warn("Failed to deliver group message", map[string]interface{}{
				"group":    groupRef,
				"memberID": memberID,
				"error":    err.Error(),
			})
		}
	}

	return nil
}
