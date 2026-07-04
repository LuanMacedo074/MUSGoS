package mus_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/types/lingo"
)

func TestSender_SendMessage_DirectUser(t *testing.T) {
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	connWriter := &testutil.MockConnectionWriter{}
	sender := mus.NewSender(connWriter, sessionStore, logger, nil, false, "faria")

	sessionStore.RegisterConnection("user2", "192.168.1.2")

	err := sender.SendMessage("user1", "user2", "chat", lingo.NewLString("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(connWriter.Writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(connWriter.Writes))
	}
	if connWriter.Writes[0].ClientID != "user2" {
		t.Errorf("write clientID = %q, want %q", connWriter.Writes[0].ClientID, "user2")
	}
}

func TestSender_SendMessage_GroupBroadcast(t *testing.T) {
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	connWriter := &testutil.MockConnectionWriter{}
	sender := mus.NewSender(connWriter, sessionStore, logger, nil, false, "faria")

	// Put sender in a movie
	sessionStore.JoinRoom("movie:myMovie", "user1")

	// Put two users in the group
	sessionStore.JoinRoom("myMovie:@AllUsers", "user1")
	sessionStore.JoinRoom("myMovie:@AllUsers", "user2")

	err := sender.SendMessage("user1", "@AllUsers", "chat", lingo.NewLString("broadcast"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(connWriter.Writes) != 2 {
		t.Fatalf("expected 2 writes, got %d", len(connWriter.Writes))
	}
}

func TestSender_SendMessage_GroupBroadcast_SenderNotInMovie(t *testing.T) {
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	connWriter := &testutil.MockConnectionWriter{}

	// With no default movie configured, a sender outside any movie errors.
	sender := mus.NewSender(connWriter, sessionStore, logger, nil, false, "")
	err := sender.SendMessage("user1", "@AllUsers", "chat", lingo.NewLString("broadcast"))
	if err == nil {
		t.Error("expected error when sender is not in any movie and no default movie is set")
	}

	// With a default movie, system senders (jobs, system.script) resolve the
	// group through it.
	sessionStore.JoinRoom("faria:@AllUsers", "user1")
	sender = mus.NewSender(connWriter, sessionStore, logger, nil, false, "faria")
	if err := sender.SendMessage("system.jobs", "@AllUsers", "chat", lingo.NewLString("broadcast")); err != nil {
		t.Fatalf("unexpected error via default-movie fallback: %v", err)
	}
	if len(connWriter.Writes) != 1 {
		t.Fatalf("expected 1 write via fallback, got %d", len(connWriter.Writes))
	}
}
