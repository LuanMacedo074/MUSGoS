package mus_test

import (
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

// setupSystemCommandsService creates a SystemService with a logged-in user in a movie.
func setupSystemCommandsService(t *testing.T) (*mus.SystemService, *testutil.MockSessionStore) {
	t.Helper()
	db := &testutil.MockDBAdapter{}
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("conn-1", "192.168.1.10")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}

	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 40, nil, nil, nil)

	// Logon user1 to join movie "testMovie"
	logonMsg := buildLogonMsg("user1", "")
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

func buildSystemMsg(subject string, content lingo.LValue) *smus.MUSMessage {
	return &smus.MUSMessage{
		Subject:  smus.MUSMsgHeaderString{Length: len(subject), Value: subject},
		SenderID: smus.MUSMsgHeaderString{Length: 5, Value: "user1"},
		RecptID: smus.MUSMsgHeaderStringList{
			Count:   1,
			Strings: []smus.MUSMsgHeaderString{{Length: 6, Value: "System"}},
		},
		MsgContent: content,
	}
}

// --- Server commands ---

func TestSystemCommand_ServerGetVersion(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.server.getVersion", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	s, ok := resp.MsgContent.(*lingo.LString)
	if !ok {
		t.Fatalf("content type = %T, want *lingo.LString", resp.MsgContent)
	}
	if s.Value != mus.ServerVersion {
		t.Errorf("version = %q, want %q", s.Value, mus.ServerVersion)
	}
}

func TestSystemCommand_ServerGetTime(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	before := time.Now().Unix()
	resp, err := svc.Handle("user1", buildSystemMsg("system.server.getTime", lingo.NewLVoid()))
	after := time.Now().Unix()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	ts := resp.MsgContent.ToInteger()
	if int64(ts) < before || int64(ts) > after {
		t.Errorf("timestamp %d not in range [%d, %d]", ts, before, after)
	}
}

func TestSystemCommand_ServerGetUserCount(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.server.getUserCount", lingo.NewLVoid()))
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

func TestSystemCommand_ServerGetMovieCount(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.server.getMovieCount", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	count := resp.MsgContent.ToInteger()
	if count != 1 {
		t.Errorf("movie count = %d, want 1", count)
	}
}

func TestSystemCommand_ServerGetMovies(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.server.getMovies", lingo.NewLVoid()))
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
	if len(list.Values) != 1 {
		t.Fatalf("movie list length = %d, want 1", len(list.Values))
	}
	if lingo.StringValue(list.Values[0]) != "testMovie" {
		t.Errorf("movie = %q, want %q", lingo.StringValue(list.Values[0]), "testMovie")
	}
}

// --- Movie commands ---

func TestSystemCommand_MovieGetUserCount(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.movie.getUserCount", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	count := resp.MsgContent.ToInteger()
	if count != 1 {
		t.Errorf("movie user count = %d, want 1", count)
	}
}

func TestSystemCommand_MovieGetGroups(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.movie.getGroups", lingo.NewLVoid()))
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
	// Should have at least @AllUsers
	found := false
	for _, v := range list.Values {
		if lingo.StringValue(v) == "@AllUsers" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected @AllUsers in group list, got %v", list.Values)
	}
}

func TestSystemCommand_MovieGetGroupCount(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.movie.getGroupCount", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	count := resp.MsgContent.ToInteger()
	if count < 1 {
		t.Errorf("group count = %d, want >= 1", count)
	}
}

func TestSystemCommand_MovieNotInMovie(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("lonely", "10.0.0.1")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}
	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 40, nil, nil, nil)

	resp, err := svc.Handle("lonely", buildSystemMsg("system.movie.getUserCount", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrServerInternalError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrServerInternalError)
	}
}

// --- Group commands ---

func TestSystemCommand_GroupJoinAndLeave(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)

	// Join a group
	resp, err := svc.Handle("user1", buildSystemMsg("system.group.join", lingo.NewLString("chatRoom")))
	if err != nil {
		t.Fatalf("join error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("join ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}

	// Verify user is in the group
	resp, err = svc.Handle("user1", buildSystemMsg("system.group.getUsers", lingo.NewLString("chatRoom")))
	if err != nil {
		t.Fatalf("getUsers error: %v", err)
	}
	list := resp.MsgContent.(*lingo.LList)
	found := false
	for _, v := range list.Values {
		if lingo.StringValue(v) == "user1" {
			found = true
		}
	}
	if !found {
		t.Errorf("user1 not found in chatRoom members")
	}

	// Leave the group
	resp, err = svc.Handle("user1", buildSystemMsg("system.group.leave", lingo.NewLString("chatRoom")))
	if err != nil {
		t.Fatalf("leave error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("leave ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}

	// Verify user left
	resp, err = svc.Handle("user1", buildSystemMsg("system.group.getUserCount", lingo.NewLString("chatRoom")))
	if err != nil {
		t.Fatalf("getUserCount error: %v", err)
	}
	count := resp.MsgContent.ToInteger()
	if count != 0 {
		t.Errorf("chatRoom user count = %d after leave, want 0", count)
	}
}

func TestSystemCommand_GroupAttributes(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)

	// Join a group first
	svc.Handle("user1", buildSystemMsg("system.group.join", lingo.NewLString("myGroup")))

	// Set attribute
	setPlist := lingo.NewLPropList()
	setPlist.AddElement(lingo.NewLSymbol("group"), lingo.NewLString("myGroup"))
	setPlist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("color"))
	setPlist.AddElement(lingo.NewLSymbol("value"), lingo.NewLString("blue"))
	resp, err := svc.Handle("user1", buildSystemMsg("system.group.setAttribute", setPlist))
	if err != nil {
		t.Fatalf("setAttribute error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("setAttribute ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}

	// Get attribute
	getPlist := lingo.NewLPropList()
	getPlist.AddElement(lingo.NewLSymbol("group"), lingo.NewLString("myGroup"))
	getPlist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("color"))
	resp, err = svc.Handle("user1", buildSystemMsg("system.group.getAttribute", getPlist))
	if err != nil {
		t.Fatalf("getAttribute error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("getAttribute ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	if lingo.StringValue(resp.MsgContent) != "blue" {
		t.Errorf("attribute value = %q, want %q", lingo.StringValue(resp.MsgContent), "blue")
	}

	// Get attribute names
	resp, err = svc.Handle("user1", buildSystemMsg("system.group.getAttributeNames", lingo.NewLString("myGroup")))
	if err != nil {
		t.Fatalf("getAttributeNames error: %v", err)
	}
	namesList := resp.MsgContent.(*lingo.LList)
	if len(namesList.Values) != 1 || lingo.StringValue(namesList.Values[0]) != "color" {
		t.Errorf("attribute names = %v, want [color]", namesList.Values)
	}

	// Delete attribute
	delPlist := lingo.NewLPropList()
	delPlist.AddElement(lingo.NewLSymbol("group"), lingo.NewLString("myGroup"))
	delPlist.AddElement(lingo.NewLSymbol("attribute"), lingo.NewLString("color"))
	resp, err = svc.Handle("user1", buildSystemMsg("system.group.deleteAttribute", delPlist))
	if err != nil {
		t.Fatalf("deleteAttribute error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("deleteAttribute ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}

	// Verify attribute deleted
	resp, err = svc.Handle("user1", buildSystemMsg("system.group.getAttributeNames", lingo.NewLString("myGroup")))
	if err != nil {
		t.Fatalf("getAttributeNames error: %v", err)
	}
	namesList = resp.MsgContent.(*lingo.LList)
	if len(namesList.Values) != 0 {
		t.Errorf("attribute names after delete = %v, want empty", namesList.Values)
	}
}

// --- User commands ---

func TestSystemCommand_UserGetAddress(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)
	resp, err := svc.Handle("user1", buildSystemMsg("system.user.getAddress", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	ip := lingo.StringValue(resp.MsgContent)
	// After logon, RegisterConnection(userID, connectionID) is called,
	// so the IP in the mock becomes the original connection ID "conn-1".
	if ip != "conn-1" {
		t.Errorf("IP = %q, want %q", ip, "conn-1")
	}
}

func TestSystemCommand_UserGetGroups(t *testing.T) {
	svc, _ := setupSystemCommandsService(t)

	// Join a custom group
	svc.Handle("user1", buildSystemMsg("system.group.join", lingo.NewLString("lobby")))

	resp, err := svc.Handle("user1", buildSystemMsg("system.user.getGroups", lingo.NewLVoid()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Fatalf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
	list := resp.MsgContent.(*lingo.LList)
	groups := map[string]bool{}
	for _, v := range list.Values {
		groups[lingo.StringValue(v)] = true
	}
	if !groups["@AllUsers"] {
		t.Error("expected @AllUsers in user groups")
	}
	if !groups["lobby"] {
		t.Error("expected lobby in user groups")
	}
}

func TestSystemCommand_UserDelete_Admin(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("admin", "10.0.0.1")
	sessionStore.RegisterConnection("victim", "10.0.0.2")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}
	cmdLevels := map[string]int{"system.user.delete": 80}
	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 80, cmdLevels, nil, nil)

	// Logon admin (defaultUserLevel=80)
	logonMsg := buildLogonMsg("admin", "")
	resp, err := svc.Handle("admin", logonMsg)
	if err != nil || resp.ErrCode != smus.ErrNoError {
		t.Fatalf("admin logon failed")
	}

	// Delete victim
	msg := buildSystemMsg("system.user.delete", lingo.NewLString("victim"))
	msg.SenderID = smus.MUSMsgHeaderString{Length: 5, Value: "admin"}
	resp, err = svc.Handle("admin", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrNoError {
		t.Errorf("ErrCode = %d, want %d", resp.ErrCode, smus.ErrNoError)
	}
}

func TestSystemCommand_UserDelete_NonAdmin(t *testing.T) {
	db := &testutil.MockDBAdapter{}
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("conn-1", "192.168.1.10")
	movieManager := mus.NewMovieManager(sessionStore, logger)
	groupManager := mus.NewGroupManager(sessionStore, logger)
	connWriter := &testutil.MockConnectionWriter{}
	cmdLevels := map[string]int{"system.user.delete": 80}
	svc := mus.NewSystemService(db, sessionStore, nil, logger, movieManager, groupManager, connWriter, "none", 40, cmdLevels, nil, nil)

	// Logon user1 (level 40) to join movie
	logonMsg := buildLogonMsg("user1", "")
	logonMsg.MsgContent.(*lingo.LList).Values[0] = lingo.NewLString("testMovie")
	svc.Handle("conn-1", logonMsg)

	msg := buildSystemMsg("system.user.delete", lingo.NewLString("someone"))
	resp, err := svc.Handle("user1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ErrCode != smus.ErrInvalidServerCommand {
		t.Errorf("ErrCode = %d, want %d (non-admin should be rejected)", resp.ErrCode, smus.ErrInvalidServerCommand)
	}
}
