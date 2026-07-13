package services_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/domain/services"
)

func creds(movieID, userID, password string) *services.LogonCredentials {
	return &services.LogonCredentials{MovieID: movieID, UserID: userID, Password: password}
}

func TestLogonService_NoneMode_AcceptsAndRegistersSession(t *testing.T) {
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("client-1", "192.168.1.1")

	var remappedOld, remappedNew string
	connWriter := &testutil.MockConnectionWriter{
		RemapFn: func(oldID, newID string) {
			remappedOld = oldID
			remappedNew = newID
		},
	}

	svc := services.NewLogonService(&testutil.MockDBAdapter{}, sessions, connWriter, &testutil.MockLogger{}, "none", 40)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		Credentials:  creds("lobby", "alice", "whatever"),
	})

	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK", res.Code)
	}
	if res.UserID != "alice" || res.UserLevel != 40 || res.MovieID != "lobby" {
		t.Errorf("result = %+v, want UserID=alice UserLevel=40 MovieID=lobby", res)
	}
	if remappedOld != "client-1" || remappedNew != "alice" {
		t.Errorf("RemapClientID(%q, %q), want (client-1, alice)", remappedOld, remappedNew)
	}

	// The session was re-registered under the userID, preserving the real IP.
	conn, err := sessions.GetConnection("alice")
	if err != nil || conn == nil {
		t.Fatalf("GetConnection(alice) = %v, %v; want a live session", conn, err)
	}
	if conn.IP != "192.168.1.1" {
		t.Errorf("re-registered IP = %q, want the original 192.168.1.1", conn.IP)
	}
	if old, _ := sessions.GetConnection("client-1"); old != nil {
		t.Errorf("connection-id session should be unregistered, still present: %+v", old)
	}

	// The user level was stamped into the session for permission checks.
	val, err := sessions.GetUserAttribute("alice", services.UserLevelAttribute)
	if err != nil {
		t.Fatalf("GetUserAttribute(%s) error: %v", services.UserLevelAttribute, err)
	}
	if got := val.ToInteger(); got != 40 {
		t.Errorf("session %s = %d, want 40", services.UserLevelAttribute, got)
	}
}

func TestLogonService_RefusesWhenUserIDAlreadyConnected(t *testing.T) {
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("client-2", "10.0.0.2")
	sessions.RegisterConnection("alice", "10.0.0.9") // alice already has a live session

	svc := services.NewLogonService(&testutil.MockDBAdapter{}, sessions, &testutil.MockConnectionWriter{}, &testutil.MockLogger{}, "none", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-2",
		SenderID:     "client-2",
		Credentials:  creds("", "alice", ""),
	})

	if res.Code != services.LogonRefused {
		t.Fatalf("Code = %v, want LogonRefused (takeover guard)", res.Code)
	}
	if res.UserID != "alice" {
		t.Errorf("UserID = %q, want alice (response recipient)", res.UserID)
	}
}

func TestLogonService_RefusesWhenRemapLosesRace(t *testing.T) {
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("client-3", "10.0.0.3")

	refuse := false
	connWriter := &testutil.MockConnectionWriter{RemapResult: &refuse}

	svc := services.NewLogonService(&testutil.MockDBAdapter{}, sessions, connWriter, &testutil.MockLogger{}, "none", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-3",
		SenderID:     "client-3",
		Credentials:  creds("", "bob", ""),
	})

	if res.Code != services.LogonRefused {
		t.Fatalf("Code = %v, want LogonRefused (remap race)", res.Code)
	}
}

func TestLogonService_SameIDAsConnection_SkipsTakeoverGuard(t *testing.T) {
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("alice", "10.0.0.4")

	svc := services.NewLogonService(&testutil.MockDBAdapter{}, sessions, &testutil.MockConnectionWriter{}, &testutil.MockLogger{}, "none", 20)

	// Logging on with a userID equal to the connection id must not trip the
	// takeover guard against its own session.
	res := svc.Logon(services.LogonRequest{
		ConnectionID: "alice",
		SenderID:     "alice",
		Credentials:  creds("", "alice", ""),
	})

	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK", res.Code)
	}
}
