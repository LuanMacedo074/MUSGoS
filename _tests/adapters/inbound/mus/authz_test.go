package mus_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

// newNonAdminService logs on `userID` at level 20 (below the DBAdmin bar of 80)
// and returns the service. dbCommandLevels is defined in db_commands_test.go.
func newNonAdminService(t *testing.T, db *testutil.MockDBAdapter, userID string) *mus.SystemService {
	t.Helper()
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("conn-1", "192.168.1.10")
	mm := mus.NewMovieManager(sessionStore, logger)
	gm := mus.NewGroupManager(sessionStore, logger)
	cw := &testutil.MockConnectionWriter{}
	svc := mus.NewSystemService(db, sessionStore, nil, logger, mm, gm, cw, "none", 20, dbCommandLevels, nil, nil)

	logon := buildLogonMsg(userID, "")
	logon.MsgContent.(*lingo.LList).Values[0] = lingo.NewLString("m")
	if resp, err := svc.Handle("conn-1", logon); err != nil || resp.ErrCode != smus.ErrNoError {
		t.Fatalf("logon setup failed: err=%v errCode=%d", err, resp.ErrCode)
	}
	return svc
}

func dbPlayerGetMsg(targetUser string) *smus.MUSMessage {
	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("app"))
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString(targetUser))
	plist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("gold"))
	return buildDBMsg("DBPlayer.getAttribute", plist)
}

// H2: a non-admin caller must not read another user's player data.
func TestDBPlayer_CrossUser_DeniedForNonAdmin(t *testing.T) {
	getCalled := false
	db := &testutil.MockDBAdapter{
		GetPlayerAttributeFunc: func(app, user, attr string) (lingo.LValue, error) {
			getCalled = true
			return lingo.NewLInteger(999), nil
		},
	}
	svc := newNonAdminService(t, db, "lowuser")

	resp, err := svc.Handle("lowuser", dbPlayerGetMsg("victim"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidServerCommand {
		t.Errorf("cross-user access ErrCode = %d, want %d (denied)", resp.ErrCode, smus.ErrInvalidServerCommand)
	}
	if getCalled {
		t.Error("DB must not be queried when cross-user access is denied")
	}
}

// H2: a non-admin caller may still access their OWN player data.
func TestDBPlayer_OwnData_AllowedForNonAdmin(t *testing.T) {
	getCalled := false
	db := &testutil.MockDBAdapter{
		GetPlayerAttributeFunc: func(app, user, attr string) (lingo.LValue, error) {
			getCalled = true
			return lingo.NewLInteger(42), nil
		},
	}
	svc := newNonAdminService(t, db, "lowuser")

	resp, err := svc.Handle("lowuser", dbPlayerGetMsg("lowuser"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("own-data access ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if !getCalled {
		t.Error("expected own-data access to reach the DB")
	}
}

// H3: a Logon for a userID that already has a live session must be refused.
func TestLogon_DuplicateUserID_Rejected(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("conn-1", "1.1.1.1")
	sessionStore.RegisterConnection("conn-2", "2.2.2.2")
	mm := mus.NewMovieManager(sessionStore, logger)
	gm := mus.NewGroupManager(sessionStore, logger)
	cw := &testutil.MockConnectionWriter{}
	svc := mus.NewSystemService(db, sessionStore, nil, logger, mm, gm, cw, "none", 40, nil, nil, nil)

	resp1, err := svc.Handle("conn-1", buildLogonMsg("dupuser", ""))
	if err != nil || resp1.ErrCode != smus.ErrNoError {
		t.Fatalf("first logon failed: err=%v errCode=%d", err, resp1.ErrCode)
	}

	resp2, err := svc.Handle("conn-2", buildLogonMsg("dupuser", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.ErrCode != smus.ErrConnectionRefused {
		t.Errorf("duplicate logon ErrCode = %d, want %d (refused)", resp2.ErrCode, smus.ErrConnectionRefused)
	}
}
