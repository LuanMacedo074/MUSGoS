package mus_test

import (
	"sort"
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
)

func setupMovieManager() (*mus.MovieManager, *testutil.MockSessionStore) {
	logger := &testutil.MockLogger{}
	sessionStore := testutil.NewMockSessionStore()
	sessionStore.RegisterConnection("user1", "192.168.1.1")
	sessionStore.RegisterConnection("user2", "192.168.1.2")
	mm := mus.NewMovieManager(sessionStore, logger)
	return mm, sessionStore
}

func TestMovieManager_JoinMovie_CreatesMovie(t *testing.T) {
	mm, _ := setupMovieManager()

	if err := mm.JoinMovie("lobby", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := mm.GetMovie("lobby")
	if !ok {
		t.Fatal("expected movie 'lobby' to exist")
	}

	movies := mm.GetMovies()
	if len(movies) != 1 || movies[0] != "lobby" {
		t.Errorf("GetMovies() = %v, want [lobby]", movies)
	}
}

func TestMovieManager_JoinMovie_AutoJoinsAllUsers(t *testing.T) {
	mm, sessionStore := setupMovieManager()

	if err := mm.JoinMovie("lobby", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check user is in @AllUsers group room
	members, err := sessionStore.GetRoomMembers("lobby:@AllUsers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 1 || members[0] != "user1" {
		t.Errorf("@AllUsers members = %v, want [user1]", members)
	}

	// Check movie has @AllUsers group
	movie, _ := mm.GetMovie("lobby")
	_, ok := movie.GetGroup("@AllUsers")
	if !ok {
		t.Fatal("expected @AllUsers group to exist in movie")
	}
}

func TestMovieManager_JoinMovie_ExistingMovie(t *testing.T) {
	mm, _ := setupMovieManager()

	mm.JoinMovie("lobby", "user1")
	if err := mm.JoinMovie("lobby", "user2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	users, err := mm.GetMovieUsers("lobby")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(users)
	if len(users) != 2 || users[0] != "user1" || users[1] != "user2" {
		t.Errorf("GetMovieUsers() = %v, want [user1 user2]", users)
	}

	count, err := mm.GetMovieUserCount("lobby")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("GetMovieUserCount() = %d, want 2", count)
	}
}

func TestMovieManager_LeaveMovie_RemovesUser(t *testing.T) {
	mm, _ := setupMovieManager()

	mm.JoinMovie("lobby", "user1")
	mm.JoinMovie("lobby", "user2")

	if err := mm.LeaveMovie("lobby", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	users, _ := mm.GetMovieUsers("lobby")
	if len(users) != 1 || users[0] != "user2" {
		t.Errorf("after leave, users = %v, want [user2]", users)
	}
}

func TestMovieManager_LeaveMovie_DestroysEmptyMovie(t *testing.T) {
	mm, _ := setupMovieManager()

	mm.JoinMovie("lobby", "user1")
	mm.LeaveMovie("lobby", "user1")

	_, ok := mm.GetMovie("lobby")
	if ok {
		t.Fatal("expected movie 'lobby' to be destroyed after last user left")
	}

	movies := mm.GetMovies()
	if len(movies) != 0 {
		t.Errorf("GetMovies() = %v, want []", movies)
	}
}

func TestMovieManager_LeaveMovie_LeavesAllGroups(t *testing.T) {
	mm, sessionStore := setupMovieManager()

	mm.JoinMovie("lobby", "user1")

	// User is auto-joined to @AllUsers; verify they leave it
	if err := mm.LeaveMovie("lobby", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	members, _ := sessionStore.GetRoomMembers("lobby:@AllUsers")
	if len(members) != 0 {
		t.Errorf("after leave, @AllUsers members = %v, want []", members)
	}
}

func TestMovieManager_GetMovies(t *testing.T) {
	mm, _ := setupMovieManager()

	mm.JoinMovie("lobby", "user1")
	mm.JoinMovie("game", "user2")

	movies := mm.GetMovies()
	sort.Strings(movies)
	if len(movies) != 2 || movies[0] != "game" || movies[1] != "lobby" {
		t.Errorf("GetMovies() = %v, want [game lobby]", movies)
	}
}

func TestMovieManager_GetMovieUsers(t *testing.T) {
	mm, _ := setupMovieManager()

	mm.JoinMovie("lobby", "user1")
	mm.JoinMovie("lobby", "user2")

	users, err := mm.GetMovieUsers("lobby")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(users)
	if len(users) != 2 || users[0] != "user1" || users[1] != "user2" {
		t.Errorf("GetMovieUsers() = %v, want [user1 user2]", users)
	}
}

func TestMovieManager_LeaveMovie_NotFound(t *testing.T) {
	mm, _ := setupMovieManager()
	err := mm.LeaveMovie("nonexistent", "user1")
	if err == nil {
		t.Error("expected error for nonexistent movie")
	}
}
