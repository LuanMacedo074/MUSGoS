package mus_test

import (
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

// dbCommandLevels maps all DB commands to level 20 so the admin (level 80) can execute them.
var dbCommandLevels = map[string]int{
	"DBPlayer.getAttribute":          20,
	"DBPlayer.setAttribute":          20,
	"DBPlayer.deleteAttribute":       20,
	"DBPlayer.getAttributeNames":     20,
	"DBApplication.getAttribute":     20,
	"DBApplication.setAttribute":     20,
	"DBApplication.deleteAttribute":  20,
	"DBApplication.getAttributeNames": 20,
	"DBAdmin.createApplication":      80,
	"DBAdmin.deleteApplication":      80,
	"DBAdmin.createUser":             80,
	"DBAdmin.deleteUser":             80,
	"DBAdmin.getUserCount":           80,
	"DBAdmin.ban":                    80,
	"DBAdmin.revokeBan":              80,
}

// setupDBCommandsService creates a SystemService with a logged-in admin user (level 80).
func setupDBCommandsService(t *testing.T, db *testutil.MockDBAdapter) (*mus.SystemService, *testutil.MockSessionStore) {
	t.Helper()
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("conn-1", "192.168.1.10")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}

	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 80, dbCommandLevels, nil, nil)

	// Logon admin to join movie "testMovie"
	logonMsg := buildLogonMsg("admin", "")
	logonMsg.MsgContent.(*lingo.LList).Values[0] = lingo.NewLString("testMovie")
	resp, err := svc.Handle("conn-1", logonMsg)
	if err != nil {
		t.Fatalf("logon error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("logon failed: errCode=%d", resp.ErrCode)
	}

	return svc, sessionStore
}

func buildDBMsg(subject string, content lingo.LValue) *smus.MUSMessage {
	return &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: len(subject), Value: subject},
		SenderID: smus.MUSMsgHeaderString{Length: 5, Value: "admin"},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 6, Value: "System"}},
		},
		MsgContent: content,
	}
}

// --- DBPlayer ---

func TestDBPlayer_SetGetAttribute(t *testing.T) {
	var stored lingo.LValue
	setCalled := false
	getCalled := false
	db := &testutil.MockDBAdapter{
		SetPlayerAttributeFunc: func(app, user, attr string, value lingo.LValue) error {
			setCalled = true
			stored = value
			return nil
		},
		GetPlayerAttributeFunc: func(app, user, attr string) (lingo.LValue, error) {
			getCalled = true
			return stored, nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	// Set
	setPlist := lingo.NewLPropList()
	setPlist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	setPlist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("player1"))
	setPlist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("score"))
	setPlist.AddElement(lingo.NewLSymbol("value"), lingo.NewLInteger(100))
	resp, err := svc.Handle("admin", buildDBMsg("DBPlayer.setAttribute", setPlist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if !setCalled {
		t.Error("expected SetPlayerAttribute to be called")
	}

	// Get
	getPlist := lingo.NewLPropList()
	getPlist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	getPlist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("player1"))
	getPlist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("score"))
	resp, err = svc.Handle("admin", buildDBMsg("DBPlayer.getAttribute", getPlist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if resp.MsgContent.ToInteger() != 100 {
		t.Errorf("value = %d, want 100", resp.MsgContent.ToInteger())
	}
	if !getCalled {
		t.Error("expected GetPlayerAttribute to be called")
	}
}

func TestDBPlayer_DeleteAttribute(t *testing.T) {
	deleted := false
	db := &testutil.MockDBAdapter{
		DeletePlayerAttributeFunc: func(app, user, attr string) error {
			deleted = true
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("player1"))
	plist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("score"))
	resp, err := svc.Handle("admin", buildDBMsg("DBPlayer.deleteAttribute", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if !deleted {
		t.Error("expected DeletePlayerAttribute to be called")
	}
}

func TestDBPlayer_GetAttributeNames(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetPlayerAttributeNamesFunc: func(app, user string) ([]string, error) {
			return []string{"score", "level"}, nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("player1"))
	resp, err := svc.Handle("admin", buildDBMsg("DBPlayer.getAttributeNames", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	list, ok := resp.MsgContent.(*lingo.LList)
	if !ok {
		t.Fatalf("content type = %T, want *lingo.LList", resp.MsgContent)
	}
	if len(list.Values) != 2 {
		t.Fatalf("names count = %d, want 2", len(list.Values))
	}
}

// --- DBApplication ---

func TestDBApplication_SetGetAttribute(t *testing.T) {
	var stored lingo.LValue
	setCalled := false
	getCalled := false
	db := &testutil.MockDBAdapter{
		SetApplicationAttributeFunc: func(app, attr string, value lingo.LValue) error {
			setCalled = true
			stored = value
			return nil
		},
		GetApplicationAttributeFunc: func(app, attr string) (lingo.LValue, error) {
			getCalled = true
			return stored, nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	// Set
	setPlist := lingo.NewLPropList()
	setPlist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	setPlist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("maxPlayers"))
	setPlist.AddElement(lingo.NewLSymbol("value"), lingo.NewLInteger(50))
	resp, err := svc.Handle("admin", buildDBMsg("DBApplication.setAttribute", setPlist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if !setCalled {
		t.Error("expected SetApplicationAttribute to be called")
	}

	// Get
	getPlist := lingo.NewLPropList()
	getPlist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	getPlist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("maxPlayers"))
	resp, err = svc.Handle("admin", buildDBMsg("DBApplication.getAttribute", getPlist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if resp.MsgContent.ToInteger() != 50 {
		t.Errorf("value = %d, want 50", resp.MsgContent.ToInteger())
	}
	if !getCalled {
		t.Error("expected GetApplicationAttribute to be called")
	}
}

func TestDBApplication_DeleteAttribute(t *testing.T) {
	deleted := false
	db := &testutil.MockDBAdapter{
		DeleteApplicationAttributeFunc: func(app, attr string) error {
			deleted = true
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	plist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("maxPlayers"))
	resp, err := svc.Handle("admin", buildDBMsg("DBApplication.deleteAttribute", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if !deleted {
		t.Error("expected DeleteApplicationAttribute to be called")
	}
}

func TestDBApplication_GetAttributeNames(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetApplicationAttributeNamesFunc: func(app string) ([]string, error) {
			return []string{"maxPlayers", "version"}, nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("myApp"))
	resp, err := svc.Handle("admin", buildDBMsg("DBApplication.getAttributeNames", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	list, ok := resp.MsgContent.(*lingo.LList)
	if !ok {
		t.Fatalf("content type = %T, want *lingo.LList", resp.MsgContent)
	}
	if len(list.Values) != 2 {
		t.Fatalf("names count = %d, want 2", len(list.Values))
	}
}

// --- DBAdmin ---

func TestDBAdmin_CreateApplication(t *testing.T) {
	created := ""
	db := &testutil.MockDBAdapter{
		CreateApplicationFunc: func(appName string) error {
			created = appName
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("newApp"))
	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.createApplication", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if created != "newApp" {
		t.Errorf("created app = %q, want %q", created, "newApp")
	}
}

func TestDBAdmin_DeleteApplication(t *testing.T) {
	deleted := ""
	db := &testutil.MockDBAdapter{
		DeleteApplicationFunc: func(appName string) error {
			deleted = appName
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("oldApp"))
	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.deleteApplication", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if deleted != "oldApp" {
		t.Errorf("deleted app = %q, want %q", deleted, "oldApp")
	}
}

func TestDBAdmin_CreateUser(t *testing.T) {
	var createdUser string
	db := &testutil.MockDBAdapter{
		CreateUserFunc: func(username, passwordHash string, userLevel int) error {
			createdUser = username
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("newuser"))
	plist.AddElement(lingo.NewLSymbol("password"), lingo.NewLString("secret123"))
	plist.AddElement(lingo.NewLSymbol("userLevel"), lingo.NewLInteger(20))
	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.createUser", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if createdUser != "newuser" {
		t.Errorf("created user = %q, want %q", createdUser, "newuser")
	}
}

func TestDBAdmin_DeleteUser(t *testing.T) {
	var deletedUser string
	db := &testutil.MockDBAdapter{
		DeleteUserFunc: func(username string) error {
			deletedUser = username
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("olduser"))
	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.deleteUser", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if deletedUser != "olduser" {
		t.Errorf("deleted user = %q, want %q", deletedUser, "olduser")
	}
}

func TestDBAdmin_GetUserCount(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	svc, _ := setupDBCommandsService(t, db)

	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.getUserCount", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	count := resp.MsgContent.ToInteger()
	if count < 1 {
		t.Errorf("user count = %d, want >= 1", count)
	}
}

func TestDBAdmin_Ban(t *testing.T) {
	var bannedUserID *int64
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 42, Username: username}, nil
		},
		CreateBanFunc: func(userID *int64, ipAddress *string, reason string, expiresAt *time.Time) error {
			bannedUserID = userID
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("baduser"))
	plist.AddElement(lingo.NewLSymbol("reason"), lingo.NewLString("cheating"))
	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.ban", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if bannedUserID == nil || *bannedUserID != 42 {
		t.Errorf("banned userID = %v, want 42", bannedUserID)
	}
}

func TestDBAdmin_RevokeBan(t *testing.T) {
	var revokedBanID int64
	db := &testutil.MockDBAdapter{
		GetUserFunc: func(username string) (*ports.User, error) {
			return &ports.User{ID: 42, Username: username}, nil
		},
		GetActiveBanByUserIDFunc: func(userID int64) (*ports.Ban, error) {
			return &ports.Ban{ID: 7, UserID: &userID}, nil
		},
		RevokeBanFunc: func(banID int64) error {
			revokedBanID = banID
			return nil
		},
	}

	svc, _ := setupDBCommandsService(t, db)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("userID"), lingo.NewLString("baduser"))
	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.revokeBan", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if revokedBanID != 7 {
		t.Errorf("revoked banID = %d, want 7", revokedBanID)
	}
}

// --- Permission denied ---

func TestDBAdmin_PermissionDenied(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("conn-1", "192.168.1.10")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}

	// defaultUserLevel=20 — below the 80 required for DBAdmin commands
	cmdLevels := map[string]int{"DBAdmin.createApplication": 80}
	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 20, cmdLevels, nil, nil)

	logonMsg := buildLogonMsg("lowuser", "")
	logonMsg.MsgContent.(*lingo.LList).Values[0] = lingo.NewLString("testMovie")
	svc.Handle("conn-1", logonMsg)

	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("app"))
	msg := buildDBMsg("DBAdmin.createApplication", plist)
	msg.SenderID = smus.MUSMsgHeaderString{Length: 7, Value: "lowuser"}
	resp, err := svc.Handle("lowuser", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidServerCommand {
		t.Errorf("ErrCode = %d, want %d (permission denied)", resp.ErrCode, smus.ErrInvalidServerCommand)
	}
}

// --- Unknown command denied by default ---

func TestUnknownCommand_ReturnsInvalidServerCommand(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	svc, _ := setupDBCommandsService(t, db)

	// A command not registered in the handler map should be rejected
	plist := lingo.NewLPropList()
	plist.AddElement(lingo.NewLSymbol("application"), lingo.NewLString("app"))
	resp, err := svc.Handle("admin", buildDBMsg("DBAdmin.unknownCommand", plist))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidServerCommand {
		t.Errorf("ErrCode = %d, want %d (unregistered command)", resp.ErrCode, smus.ErrInvalidServerCommand)
	}
}

// --- Invalid content format ---

func TestDBPlayer_InvalidContent(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	svc, _ := setupDBCommandsService(t, db)

	// Send a string instead of a proplist
	resp, err := svc.Handle("admin", buildDBMsg("DBPlayer.getAttribute", lingo.NewLString("invalid")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidMessageFormat {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrInvalidMessageFormat)
	}
}
