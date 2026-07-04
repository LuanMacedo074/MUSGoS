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
	// defaultMovieID resolves @group sends from senders that are in no movie
	// (system.script, scheduler jobs). The FSOS client always logs into one
	// movie ("faria"), so system-authored group messages target its groups.
	defaultMovieID string
}

func NewSender(connWriter ports.ConnectionWriter, sessionStore ports.SessionStore, logger ports.Logger, cipher ports.Cipher, allEncrypted bool, defaultMovieID string) *Sender {
	return &Sender{
		connWriter:     connWriter,
		sessionStore:   sessionStore,
		logger:         logger,
		cipher:         cipher,
		allEncrypted:   allEncrypted,
		defaultMovieID: defaultMovieID,
	}
}

func (s *Sender) SendMessage(senderID, recipientID, subject string, content lingo.LValue) error {
	return s.SendMessageFrom(senderID, senderID, recipientID, subject, content)
}

// SendMessageFrom splits the sender's two roles: wireFrom goes on the wire as
// the protocol sender; routingSender resolves the movie for @group fan-out
// (see ports.MessageSender).
func (s *Sender) SendMessageFrom(wireFrom, routingSender, recipientID, subject string, content lingo.LValue) error {
	if strings.HasPrefix(recipientID, "@") {
		return s.deliverToGroup(wireFrom, routingSender, recipientID, subject, content)
	}

	msg := NewResponse(subject, wireFrom, []string{recipientID}, smus.ErrNoError, content)
	msgBytes := msg.GetBytes()
	if s.allEncrypted && s.cipher != nil {
		msgBytes = s.cipher.Encrypt(msgBytes)
	}
	return s.connWriter.WriteToClient(recipientID, msgBytes)
}

func (s *Sender) deliverToGroup(wireFrom, routingSender, groupRef, subject string, content lingo.LValue) error {
	// Find the movie the routing sender is in; senders that live in no movie
	// (system.script, jobs) fall back to the configured default movie.
	rooms, err := s.sessionStore.GetClientRooms(routingSender)
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
		movieID = s.defaultMovieID
	}
	if movieID == "" {
		return fmt.Errorf("sender %q is not in any movie", routingSender)
	}

	// Look up group members via session store room
	roomName := groupRoomName(movieID, groupRef)
	members, err := s.sessionStore.GetRoomMembers(roomName)
	if err != nil {
		return fmt.Errorf("failed to get group members for %s: %w", groupRef, err)
	}

	// Serialize once with the group reference as recipient, then deliver to all members
	msg := NewResponse(subject, wireFrom, []string{groupRef}, smus.ErrNoError, content)
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
