package mus_test

import (
	"sort"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/types/lingo"
)

func setupGroupManager() (*mus.GroupManager, *mus.MovieManager, *testutil.MockSessionStore) {
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("user1", "192.168.1.1")
	sessionStore.RegisterConnection("user2", "192.168.1.2")
	mm := mus.NewMovieManager(sessionStore, logger)
	gm := mus.NewGroupManager(sessionStore, logger)
	return gm, mm, sessionStore
}

func TestGroupManager_JoinGroup(t *testing.T) {
	gm, mm, _ := setupGroupManager()

	// User must be in a movie first
	mm.JoinMovie("lobby", "user1")

	if err := gm.JoinGroup("lobby", "testers", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	members, err := gm.GetGroupMembers("lobby", "testers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 1 || members[0] != "user1" {
		t.Errorf("GetGroupMembers() = %v, want [user1]", members)
	}
}

func TestGroupManager_JoinGroup_NotInMovie(t *testing.T) {
	gm, _, _ := setupGroupManager()

	err := gm.JoinGroup("lobby", "testers", "user1")
	if err == nil {
		t.Fatal("expected error when joining group without being in movie")
	}
}

func TestGroupManager_LeaveGroup(t *testing.T) {
	gm, mm, _ := setupGroupManager()

	mm.JoinMovie("lobby", "user1")
	gm.JoinGroup("lobby", "testers", "user1")

	if err := gm.LeaveGroup("lobby", "testers", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	members, _ := gm.GetGroupMembers("lobby", "testers")
	if len(members) != 0 {
		t.Errorf("after leave, members = %v, want []", members)
	}
}

func TestGroupManager_GetGroupMembers(t *testing.T) {
	gm, mm, _ := setupGroupManager()

	mm.JoinMovie("lobby", "user1")
	mm.JoinMovie("lobby", "user2")
	gm.JoinGroup("lobby", "testers", "user1")
	gm.JoinGroup("lobby", "testers", "user2")

	members, err := gm.GetGroupMembers("lobby", "testers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(members)
	if len(members) != 2 || members[0] != "user1" || members[1] != "user2" {
		t.Errorf("GetGroupMembers() = %v, want [user1 user2]", members)
	}

	count, err := gm.GetGroupMemberCount("lobby", "testers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("GetGroupMemberCount() = %d, want 2", count)
	}
}

func TestGroupManager_LeaveAllGroups(t *testing.T) {
	gm, mm, _ := setupGroupManager()

	mm.JoinMovie("lobby", "user1")
	gm.JoinGroup("lobby", "testers", "user1")
	gm.JoinGroup("lobby", "admins", "user1")

	if err := gm.LeaveAllGroups("lobby", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	members1, _ := gm.GetGroupMembers("lobby", "testers")
	members2, _ := gm.GetGroupMembers("lobby", "admins")
	// Also check @AllUsers (auto-joined by JoinMovie)
	members3, _ := gm.GetGroupMembers("lobby", "@AllUsers")

	if len(members1) != 0 {
		t.Errorf("testers members after LeaveAllGroups = %v, want []", members1)
	}
	if len(members2) != 0 {
		t.Errorf("admins members after LeaveAllGroups = %v, want []", members2)
	}
	if len(members3) != 0 {
		t.Errorf("@AllUsers members after LeaveAllGroups = %v, want []", members3)
	}
}

func TestGroup_Attributes(t *testing.T) {
	g := mus.NewGroup("testers", "lobby", false)

	// Set and get
	g.SetAttribute("color", lingo.NewLString("red"))
	val := g.GetAttribute("color")
	if s, ok := val.(*lingo.LString); !ok || s.Value != "red" {
		t.Errorf("GetAttribute('color') = %v, want 'red'", val)
	}

	// Get non-existent returns void
	val = g.GetAttribute("missing")
	if val.GetType() != lingo.VtVoid {
		t.Errorf("GetAttribute('missing') type = %d, want VtVoid", val.GetType())
	}

	// List attribute names
	g.SetAttribute("size", lingo.NewLInteger(10))
	names := g.GetAttributeNames()
	sort.Strings(names)
	if len(names) != 2 || names[0] != "color" || names[1] != "size" {
		t.Errorf("GetAttributeNames() = %v, want [color size]", names)
	}

	// Delete
	g.DeleteAttribute("color")
	names = g.GetAttributeNames()
	if len(names) != 1 || names[0] != "size" {
		t.Errorf("after delete, GetAttributeNames() = %v, want [size]", names)
	}
}
