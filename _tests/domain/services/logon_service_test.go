package services_test

import (
	"errors"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(hash)
}

func logonSvc(t *testing.T, db *testutil.MockDBAdapter, mode string, defaultLevel int) (*services.LogonService, *testutil.MockSessionStore) {
	t.Helper()
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("client-1", "192.168.1.1")
	svc := services.NewLogonService(db, sessions, &testutil.MockConnectionWriter{}, &testutil.MockLogger{}, mode, defaultLevel)
	return svc, sessions
}

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

func TestLogonService_StrictMode_UnknownUser(t *testing.T) {
	svc, _ := logonSvc(t, &testutil.MockDBAdapter{}, "strict", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		Credentials:  creds("", "ghost", "pw"),
	})

	if res.Code != services.LogonInvalidUser {
		t.Fatalf("Code = %v, want LogonInvalidUser", res.Code)
	}
	if res.UserID != "ghost" {
		t.Errorf("UserID = %q, want ghost", res.UserID)
	}
}

func TestLogonService_StrictMode_WrongPassword(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "alice", PasswordHash: hashPassword(t, "right")}, nil
		},
	}
	svc, _ := logonSvc(t, db, "strict", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		Credentials:  creds("", "alice", "wrong"),
	})

	if res.Code != services.LogonInvalidPassword {
		t.Fatalf("Code = %v, want LogonInvalidPassword", res.Code)
	}
}

func TestLogonService_StrictMode_BannedUser(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 7, Username: "alice", PasswordHash: hashPassword(t, "pw")}, nil
		},
		GetActiveBanByUserIDFunc: func(userID int64) (*ports.Ban, error) {
			return &ports.Ban{ID: 1, Reason: "cheating"}, nil
		},
	}
	svc, _ := logonSvc(t, db, "strict", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		Credentials:  creds("", "alice", "pw"),
	})

	if res.Code != services.LogonRefused {
		t.Fatalf("Code = %v, want LogonRefused (banned)", res.Code)
	}
}

func TestLogonService_StrictMode_Success_UsesAccountLevel(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "alice", PasswordHash: hashPassword(t, "pw"), UserLevel: 80}, nil
		},
	}
	svc, sessions := logonSvc(t, db, "strict", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		Credentials:  creds("lobby", "alice", "pw"),
	})

	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK", res.Code)
	}
	if res.UserLevel != 80 {
		t.Errorf("UserLevel = %d, want the account's 80, not the default 20", res.UserLevel)
	}
	val, err := sessions.GetUserAttribute("alice", services.UserLevelAttribute)
	if err != nil || val.ToInteger() != 80 {
		t.Errorf("session %s = %v (err %v), want 80", services.UserLevelAttribute, val, err)
	}
}

func TestLogonService_StrictMode_UnparseableCredentialsRefused(t *testing.T) {
	svc, _ := logonSvc(t, &testutil.MockDBAdapter{}, "strict", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "wire-sender",
		ParseErr:     errors.New("not a list"),
	})

	if res.Code != services.LogonBadCredentialsFormat {
		t.Fatalf("Code = %v, want LogonBadCredentialsFormat", res.Code)
	}
	if res.UserID != "wire-sender" {
		t.Errorf("UserID = %q, want the wire sender (response recipient)", res.UserID)
	}
}

func TestLogonService_OpenMode_UnparseableCredentialsFallsBackToSender(t *testing.T) {
	svc, sessions := logonSvc(t, &testutil.MockDBAdapter{}, "open", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		ParseErr:     errors.New("not a list"),
	})

	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK (fallback to sender identity)", res.Code)
	}
	if res.UserID != "client-1" {
		t.Errorf("UserID = %q, want the sender fallback client-1", res.UserID)
	}
	if conn, _ := sessions.GetConnection("client-1"); conn == nil {
		t.Error("expected a live session under the fallback identity")
	}
}

func TestLogonService_OpenMode_UnknownUserAccepted(t *testing.T) {
	svc, _ := logonSvc(t, &testutil.MockDBAdapter{}, "open", 25)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		Credentials:  creds("", "newplayer", "whatever"),
	})

	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK (open mode accepts unknown users)", res.Code)
	}
	if res.UserLevel != 25 {
		t.Errorf("UserLevel = %d, want the default 25", res.UserLevel)
	}
}

func TestLogonService_OpenMode_DatabaseErrorIsInternal(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return nil, errors.New("connection reset")
		},
	}
	svc, _ := logonSvc(t, db, "open", 20)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1",
		SenderID:     "client-1",
		Credentials:  creds("", "alice", "pw"),
	})

	if res.Code != services.LogonInternalError {
		t.Fatalf("Code = %v, want LogonInternalError", res.Code)
	}
}

func TestLogonService_OpenMode_ExistingUserMustAuthenticate(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "alice", PasswordHash: hashPassword(t, "right"), UserLevel: 60}, nil
		},
	}
	svc, _ := logonSvc(t, db, "open", 20)

	if res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("", "alice", "wrong"),
	}); res.Code != services.LogonInvalidPassword {
		t.Fatalf("wrong password: Code = %v, want LogonInvalidPassword", res.Code)
	}

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("", "alice", "right"),
	})
	if res.Code != services.LogonOK {
		t.Fatalf("right password: Code = %v, want LogonOK", res.Code)
	}
	if res.UserLevel != 60 {
		t.Errorf("UserLevel = %d, want the account's 60", res.UserLevel)
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

// tokenLogonSvc builds a "token"-mode LogonService backed by a real
// cache-backed token store, and returns both so a test can mint tokens.
func tokenLogonSvc(t *testing.T) (*services.LogonService, ports.TokenStore, *testutil.MockSessionStore) {
	t.Helper()
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("client-1", "192.168.1.1")
	store := outbound.NewCacheTokenStore(outbound.NewMemoryCache(), time.Minute)
	svc := services.NewLogonService(
		&testutil.MockDBAdapter{}, sessions, &testutil.MockConnectionWriter{}, &testutil.MockLogger{},
		"token", 20, services.WithTokenStore(store),
	)
	return svc, store, sessions
}

func TestLogonService_TokenMode_ValidToken_BindsCarriedCharacter(t *testing.T) {
	svc, store, sessions := tokenLogonSvc(t)

	token, err := store.Issue(ports.TokenPayload{CharacterName: "Frodo", UserLevel: 55})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// The client sends Logon(userID = character_name, password = token).
	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("shire", "Frodo", token),
	})

	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK", res.Code)
	}
	if res.UserID != "Frodo" {
		t.Errorf("UserID = %q, want the token's Frodo", res.UserID)
	}
	if res.UserLevel != 55 {
		t.Errorf("UserLevel = %d, want the token's 55", res.UserLevel)
	}
	if res.MovieID != "shire" {
		t.Errorf("MovieID = %q, want shire", res.MovieID)
	}
	// Session is registered under the carried character with the carried level.
	if conn, err := sessions.GetConnection("Frodo"); err != nil || conn == nil {
		t.Fatalf("GetConnection(Frodo) = %v, %v; want a live session", conn, err)
	}
	if lvl, _ := sessions.GetUserAttribute("Frodo", services.UserLevelAttribute); lvl == nil || lvl.ToInteger() != 55 {
		t.Errorf("stamped level = %v, want 55", lvl)
	}
}

func TestLogonService_TokenMode_SharedCacheInterop(t *testing.T) {
	// Mirrors main.go: the login script's token store and the Logon service's
	// token store are separate instances over the SAME cache. A token issued by
	// one must validate through the other.
	cache := outbound.NewMemoryCache()
	issuer := outbound.NewCacheTokenStore(cache, time.Minute)    // stands in for the engine/login side
	validator := outbound.NewCacheTokenStore(cache, time.Minute) // the Logon side

	token, err := issuer.Issue(ports.TokenPayload{CharacterName: "Merry", UserLevel: 30})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("client-1", "192.168.1.1")
	svc := services.NewLogonService(
		&testutil.MockDBAdapter{}, sessions, &testutil.MockConnectionWriter{}, &testutil.MockLogger{},
		"token", 20, services.WithTokenStore(validator),
	)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("", "Merry", token),
	})
	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK", res.Code)
	}
	if res.UserID != "Merry" || res.UserLevel != 30 {
		t.Errorf("result = %+v, want UserID=Merry UserLevel=30", res)
	}
}

func TestLogonService_TokenMode_TokenCharacterIsAuthoritative(t *testing.T) {
	// The token payload — not the client-supplied userID — decides the identity.
	// A client claiming "Sauron" while presenting a token for "Frodo" binds to
	// Frodo; the client name is ignored.
	svc, store, sessions := tokenLogonSvc(t)

	token, _ := store.Issue(ports.TokenPayload{CharacterName: "Frodo", UserLevel: 55})

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("", "Sauron", token),
	})

	if res.Code != services.LogonOK {
		t.Fatalf("Code = %v, want LogonOK", res.Code)
	}
	if res.UserID != "Frodo" {
		t.Errorf("UserID = %q, want the token's Frodo (client-claimed Sauron ignored)", res.UserID)
	}
	if conn, err := sessions.GetConnection("Frodo"); err != nil || conn == nil {
		t.Errorf("session should bind to Frodo, got %v (err %v)", conn, err)
	}
	if conn, _ := sessions.GetConnection("Sauron"); conn != nil {
		t.Errorf("no session should exist for the client-claimed Sauron, got %+v", conn)
	}
}

func TestLogonService_TokenMode_TokenIsSingleUse(t *testing.T) {
	svc, store, _ := tokenLogonSvc(t)

	token, _ := store.Issue(ports.TokenPayload{CharacterName: "Sam", UserLevel: 20})

	if res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("", "Sam", token),
	}); res.Code != services.LogonOK {
		t.Fatalf("first logon Code = %v, want LogonOK", res.Code)
	}

	// The token was consumed; replaying it must be refused.
	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-2", SenderID: "client-2",
		Credentials: creds("", "Sam", token),
	})
	if res.Code != services.LogonInvalidPassword {
		t.Errorf("replay Code = %v, want LogonInvalidPassword", res.Code)
	}
}

func TestLogonService_TokenMode_InvalidToken_Refused(t *testing.T) {
	svc, _, _ := tokenLogonSvc(t)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("", "Frodo", "not-a-real-token"),
	})
	if res.Code != services.LogonInvalidPassword {
		t.Errorf("Code = %v, want LogonInvalidPassword", res.Code)
	}
}

func TestLogonService_TokenMode_MissingCredentials_BadFormat(t *testing.T) {
	svc, _, _ := tokenLogonSvc(t)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: nil, ParseErr: errors.New("no creds"),
	})
	if res.Code != services.LogonBadCredentialsFormat {
		t.Errorf("Code = %v, want LogonBadCredentialsFormat", res.Code)
	}
}

func TestLogonService_TokenMode_NoStore_InternalError(t *testing.T) {
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection("client-1", "192.168.1.1")
	// Token mode but the store option was never supplied.
	svc := services.NewLogonService(
		&testutil.MockDBAdapter{}, sessions, &testutil.MockConnectionWriter{}, &testutil.MockLogger{},
		"token", 20,
	)

	res := svc.Logon(services.LogonRequest{
		ConnectionID: "client-1", SenderID: "client-1",
		Credentials: creds("", "Frodo", "whatever"),
	})
	if res.Code != services.LogonInternalError {
		t.Errorf("Code = %v, want LogonInternalError", res.Code)
	}
}
