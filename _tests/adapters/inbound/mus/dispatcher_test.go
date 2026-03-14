package mus_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func newTestDispatcher(scriptEngine ports.ScriptEngine) (*mus.Dispatcher, *testutil.MockConnectionWriter, *testutil.MockSessionStore) {
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	connWriter := &testutil.MockConnectionWriter{}
	sender := mus.NewSender(connWriter, sessionStore, logger, nil, false)
	systemService := mus.NewSystemService(nil, sessionStore, nil, logger, nil, nil, connWriter, "none", 40, nil, nil, nil)
	dispatcher := mus.NewDispatcher(logger, scriptEngine, systemService, sender, nil)
	return dispatcher, connWriter, sessionStore
}

func TestDispatcher_SystemRecipient_RoutesToSystemService(t *testing.T) {
	dispatcher, _, sessionStore := newTestDispatcher(nil)
	sessionStore.RegisterConnection("client-1", "192.168.1.1")

	msg := buildLogonMsg("testuser", "nopass")

	resp, err := dispatcher.Dispatch("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response from System handler")
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
}

func TestDispatcher_ScriptRecipient_ExecutesScript(t *testing.T) {
	var executedSubject string
	scriptEngine := &testutil.MockScriptEngine{
		HasScriptFunc: func(subject string) bool {
			return subject == "myScript"
		},
		ExecuteFunc: func(msg *ports.ScriptMessage) (*ports.ScriptResult, error) {
			executedSubject = msg.Subject
			return &ports.ScriptResult{Content: lingo.NewLString("result")}, nil
		},
	}

	dispatcher, _, _ := newTestDispatcher(scriptEngine)

	msg := &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: 8, Value: "myScript"},
		SenderID: smus.MUSMsgHeaderString{Length: 5, Value: "user1"},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 13, Value: "system.script"}},
		},
		MsgContent: lingo.NewLVoid(),
	}

	resp, err := dispatcher.Dispatch("user1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executedSubject != "myScript" {
		t.Errorf("executed subject = %q, want %q", executedSubject, "myScript")
	}
	if resp == nil {
		t.Fatal("expected response from script")
	}
}

func TestDispatcher_UserRecipient_SendsDirectMessage(t *testing.T) {
	dispatcher, connWriter, sessionStore := newTestDispatcher(nil)
	sessionStore.RegisterConnection("user2", "192.168.1.2")

	msg := &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: 4, Value: "chat"},
		SenderID: smus.MUSMsgHeaderString{Length: 5, Value: "user1"},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 5, Value: "user2"}},
		},
		MsgContent: lingo.NewLString("hello"),
	}

	resp, err := dispatcher.Dispatch("user1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response for user-to-user message")
	}

	if len(connWriter.Writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(connWriter.Writes))
	}
	if connWriter.Writes[0].ClientID != "user2" {
		t.Errorf("write clientID = %q, want %q", connWriter.Writes[0].ClientID, "user2")
	}
}

func TestDispatcher_GroupRecipient_BroadcastsToMembers(t *testing.T) {
	dispatcher, connWriter, sessionStore := newTestDispatcher(nil)

	// Set up sender in a movie with a group
	sessionStore.RegisterConnection("user1", "192.168.1.1")
	sessionStore.RegisterConnection("user2", "192.168.1.2")
	sessionStore.JoinRoom("movie:testMovie", "user1")
	sessionStore.JoinRoom("movie:testMovie", "user2")
	sessionStore.JoinRoom("testMovie:@AllUsers", "user1")
	sessionStore.JoinRoom("testMovie:@AllUsers", "user2")

	msg := &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: 4, Value: "chat"},
		SenderID: smus.MUSMsgHeaderString{Length: 5, Value: "user1"},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 9, Value: "@AllUsers"}},
		},
		MsgContent: lingo.NewLString("hello all"),
	}

	resp, err := dispatcher.Dispatch("user1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response for group broadcast")
	}

	if len(connWriter.Writes) != 2 {
		t.Fatalf("expected 2 writes (both members), got %d", len(connWriter.Writes))
	}

	receivedIDs := map[string]bool{}
	for _, w := range connWriter.Writes {
		receivedIDs[w.ClientID] = true
	}
	if !receivedIDs["user1"] {
		t.Error("expected user1 to receive group broadcast")
	}
	if !receivedIDs["user2"] {
		t.Error("expected user2 to receive group broadcast")
	}
}

func TestDispatcher_NoRecipients_ReturnsError(t *testing.T) {
	dispatcher, _, _ := newTestDispatcher(nil)

	msg := &smus.MUSMessage{
		Subject:    smus.MUSMsgHeaderString{Length: 4, Value: "test"},
		SenderID:   smus.MUSMsgHeaderString{Length: 5, Value: "user1"},
		MsgContent: lingo.NewLVoid(),
	}

	_, err := dispatcher.Dispatch("user1", msg)
	if err == nil {
		t.Error("expected error for message with no recipients")
	}
}
