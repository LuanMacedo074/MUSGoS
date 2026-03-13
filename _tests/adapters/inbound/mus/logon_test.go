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

func setupLogonService(db *testutil.MockDBAdapter, authMode string) *mus.LogonService {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("client-1", "192.168.1.1")
	return mus.NewLogonService(db, sessionStore, cipher, logger, authMode, 40)
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

func buildLogonMsgWithList(userID, password string) *smus.MUSMessage {
	list := lingo.NewLList()
	list.Values = []lingo.LValue{
		lingo.NewLString("movieID"),
		lingo.NewLString(userID),
		lingo.NewLString(password),
	}
	return &smus.MUSMessage{
		Subject:    smus.MUSMsgHeaderString{Length: 5, Value: "Logon"},
		SenderID:   smus.MUSMsgHeaderString{Length: len(userID), Value: userID},
		MsgContent: list,
	}
}

func buildLogonMsgWithPropList(userID, password string) *smus.MUSMessage {
	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString(userID))
	plist.AddElement(lingo.NewLSymbol("password"), lingo.NewLString(password))
	return &smus.MUSMessage{
		Subject:    smus.MUSMsgHeaderString{Length: 5, Value: "Logon"},
		SenderID:   smus.MUSMsgHeaderString{Length: len(userID), Value: userID},
		MsgContent: plist,
	}
}

func TestLogonService_Success_LList(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
	}

	svc := setupLogonService(db, "strict")
	msg := buildLogonMsgWithList("testuser", "secret")

	resp, err := svc.HandleLogon("client-1", msg)
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

func TestLogonService_Success_LPropList(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
	}

	svc := setupLogonService(db, "strict")
	msg := buildLogonMsgWithPropList("testuser", "secret")

	resp, err := svc.HandleLogon("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
}

func TestLogonService_UserNotFound(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	svc := setupLogonService(db, "strict")
	msg := buildLogonMsgWithList("unknown", "secret")

	resp, err := svc.HandleLogon("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidUserID {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrInvalidUserID)
	}
}

func TestLogonService_BadPassword(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("correct")}, nil
		},
	}

	svc := setupLogonService(db, "strict")
	msg := buildLogonMsgWithList("testuser", "wrong")

	resp, err := svc.HandleLogon("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidPassword {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrInvalidPassword)
	}
}

func TestLogonService_NoneMode_AcceptsAnyUser(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	svc := setupLogonService(db, "none")
	msg := buildLogonMsgWithList("randomuser", "nopass")

	resp, err := svc.HandleLogon("client-1", msg)
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

func TestLogonService_OpenMode_AcceptsUnknownUser(t *testing.T) {
	db := &testutil.MockDBAdapter{}

	svc := setupLogonService(db, "open")
	msg := buildLogonMsgWithList("newplayer", "nopass")

	resp, err := svc.HandleLogon("client-1", msg)
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

func TestLogonService_OpenMode_ValidatesExistingUser(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("correct")}, nil
		},
	}

	svc := setupLogonService(db, "open")
	msg := buildLogonMsgWithList("testuser", "wrong")

	resp, err := svc.HandleLogon("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidPassword {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrInvalidPassword)
	}
}

func TestLogonService_OpenMode_CorrectPassword(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
	}

	svc := setupLogonService(db, "open")
	msg := buildLogonMsgWithList("testuser", "secret")

	resp, err := svc.HandleLogon("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
}

func TestLogonService_BannedUser(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 1, Username: "testuser", PasswordHash: hashPassword("secret")}, nil
		},
		GetActiveBanByUserIDFunc: func(userID int64) (*ports.Ban, error) {
			return &ports.Ban{ID: 1, Reason: "cheating"}, nil
		},
	}

	svc := setupLogonService(db, "strict")
	msg := buildLogonMsgWithList("testuser", "secret")

	resp, err := svc.HandleLogon("client-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrConnectionRefused {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrConnectionRefused)
	}
}
