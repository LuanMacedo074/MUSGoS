package mus_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"

	"golang.org/x/crypto/bcrypt"
)

func setupSystemService(db *testutil.MockDBAdapter, authMode string) *mus.SystemService {
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("client-1", "192.168.1.1")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}
	return mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, authMode, 40)
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

func buildLogonMsg(userID, password string) *smus.MUSMessage {
	list := lingo.NewLList()
	list.Values = []lingo.LValue{
		lingo.NewLString("movieID"),
		lingo.NewLString(userID),
		lingo.NewLString(password),
	}
	return &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: 5, Value: "Logon"},
		SenderID: smus.MUSMsgHeaderString{Length: len(userID), Value: userID},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 6, Value: "System"}},
		},
		MsgContent: list,
	}
}

func buildLogonMsgWithPropList(userID, password string) *smus.MUSMessage {
	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString(userID))
	plist.AddElement(lingo.NewLSymbol("password"), lingo.NewLString(password))
	return &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: 5, Value: "Logon"},
		SenderID: smus.MUSMsgHeaderString{Length: len(userID), Value: userID},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 6, Value: "System"}},
		},
		MsgContent: plist,
	}
}

func TestSystemService_Logon_Success_LList(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
	}

	svc := setupSystemService(db, "strict")
	msg := buildLogonMsg("testuser", "secret")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if resp.RecptID.Strings[0].Value != "testuser" {
		t.Errorf("recipient = %q, want %q", resp.RecptID.Strings[0].Value, "testuser")
	}
}

func TestSystemService_Logon_Success_LPropList(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
	}

	svc := setupSystemService(db, "strict")
	msg := buildLogonMsgWithPropList("testuser", "secret")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
}

func TestSystemService_Logon_UserNotFound(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	svc := setupSystemService(db, "strict")
	msg := buildLogonMsg("unknown", "secret")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidUserID {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrInvalidUserID)
	}
}

func TestSystemService_Logon_BadPassword(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("correct")}, nil
		},
	}

	svc := setupSystemService(db, "strict")
	msg := buildLogonMsg("testuser", "wrong")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidPassword {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrInvalidPassword)
	}
}

func TestSystemService_Logon_NoneMode_AcceptsAnyUser(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	svc := setupSystemService(db, "none")
	msg := buildLogonMsg("randomuser", "nopass")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if resp.RecptID.Strings[0].Value != "randomuser" {
		t.Errorf("recipient = %q, want %q", resp.RecptID.Strings[0].Value, "randomuser")
	}
}

func TestSystemService_Logon_OpenMode_AcceptsUnknownUser(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	svc := setupSystemService(db, "open")
	msg := buildLogonMsg("newplayer", "nopass")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if resp.RecptID.Strings[0].Value != "newplayer" {
		t.Errorf("recipient = %q, want %q", resp.RecptID.Strings[0].Value, "newplayer")
	}
}

func TestSystemService_Logon_OpenMode_ValidatesExistingUser(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("correct")}, nil
		},
	}

	svc := setupSystemService(db, "open")
	msg := buildLogonMsg("testuser", "wrong")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidPassword {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrInvalidPassword)
	}
}

func TestSystemService_Logon_OpenMode_CorrectPassword(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
	}

	svc := setupSystemService(db, "open")
	msg := buildLogonMsg("testuser", "secret")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
}

func TestSystemService_Logon_JoinsMovie(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("client-1", "192.168.1.1")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}
	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 40)

	msg := buildLogonMsg("testuser", "nopass")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}

	members, err := sessionStore.GetRoomMembers("movie:movieID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, m := range members {
		if m == "testuser" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected testuser in movie:movieID room, got members: %v", members)
	}
}

func TestSystemService_Logon_BannedUser(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
		GetActiveBanByUserIDFunc: func(userID int64) (*ports.Ban, error) {
			return &ports.Ban{ID: 1, Reason: "cheating"}, nil
		},
	}

	svc := setupSystemService(db, "strict")
	msg := buildLogonMsg("testuser", "secret")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrConnectionRefused {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrConnectionRefused)
	}
}

func TestSystemService_Logon_RemapsClientID(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	var remappedOld, remappedNew string
	connWriter := &testutil.MockConnectionWriter{
		RemapFn: func(oldID, newID string) {
			remappedOld = oldID
			remappedNew = newID
		},
	}

	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("client-1", "192.168.1.1")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 40)

	msg := buildLogonMsg("testuser", "nopass")

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}

	if remappedOld != "client-1" || remappedNew != "testuser" {
		t.Errorf("RemapClientID(%q, %q), want (%q, %q)", remappedOld, remappedNew, "client-1", "testuser")
	}
}

func TestSystemService_UnknownCommand(t *testing.T) {
	svc := setupSystemService(&testutil.MockDBAdapter{}, "none")
	msg := &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: 7, Value: "unknown"},
		SenderID: smus.MUSMsgHeaderString{Length: 5, Value: "user1"},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 6, Value: "System"}},
		},
		MsgContent: lingo.NewLVoid(),
	}

	resp, err := svc.Handle("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response for unknown system command")
	}
}
