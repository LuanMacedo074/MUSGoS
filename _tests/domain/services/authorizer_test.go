package services_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/domain/services"
	"fsos-server/internal/domain/types/lingo"
)

func sessionWithLevel(t *testing.T, userID string, level int32) *testutil.MockSessionStore {
	t.Helper()
	sessions := testutil.NewMockSessionStore()
	sessions.RegisterConnection(userID, "10.0.0.1")
	sessions.SetUserAttribute(userID, services.UserLevelAttribute, lingo.NewLInteger(level))
	return sessions
}

func TestAuthorizer_CanRun_DeniesUnknownCommandByDefault(t *testing.T) {
	sessions := sessionWithLevel(t, "root", 100)
	authz := services.NewAuthorizer(sessions, map[string]int{"DBAdmin.createUser": 80})

	if authz.CanRun("root", "DBAdmin.dropEverything") {
		t.Error("unknown command must be denied regardless of user level (deny-by-default)")
	}
}

func TestAuthorizer_CanRun_EnforcesCommandLevel(t *testing.T) {
	levels := map[string]int{"DBAdmin.createUser": 80}

	if !services.NewAuthorizer(sessionWithLevel(t, "admin", 80), levels).CanRun("admin", "DBAdmin.createUser") {
		t.Error("level 80 must be allowed to run a level-80 command")
	}
	if services.NewAuthorizer(sessionWithLevel(t, "pleb", 79), levels).CanRun("pleb", "DBAdmin.createUser") {
		t.Error("level 79 must be denied a level-80 command")
	}
}

func TestAuthorizer_UserLevel_ZeroWhenAbsent(t *testing.T) {
	authz := services.NewAuthorizer(testutil.NewMockSessionStore(), nil)

	if got := authz.UserLevel("nobody"); got != 0 {
		t.Errorf("UserLevel(nobody) = %d, want 0", got)
	}
}

func TestAuthorizer_OwnerOrAdmin(t *testing.T) {
	levels := map[string]int{"DBAdmin.createUser": 80}

	if !services.NewAuthorizer(sessionWithLevel(t, "alice", 20), levels).OwnerOrAdmin("alice", "alice") {
		t.Error("a user must always be allowed to act on their own data")
	}
	if !services.NewAuthorizer(sessionWithLevel(t, "admin", 80), levels).OwnerOrAdmin("admin", "alice") {
		t.Error("an admin-level user must be allowed to act on another user's data")
	}
	if services.NewAuthorizer(sessionWithLevel(t, "mallory", 79), levels).OwnerOrAdmin("mallory", "alice") {
		t.Error("a below-admin user must be denied another user's data")
	}
}

func TestAuthorizer_AdminLevel_TracksConfiguredDBAdminLevel(t *testing.T) {
	// The admin threshold follows the configured DBAdmin.createUser level...
	authz := services.NewAuthorizer(sessionWithLevel(t, "mod", 90), map[string]int{"DBAdmin.createUser": 90})
	if !authz.OwnerOrAdmin("mod", "someone") {
		t.Error("level 90 must clear a configured admin level of 90")
	}

	// ...and defaults to 80 when not configured.
	if got := services.NewAuthorizer(testutil.NewMockSessionStore(), nil).AdminLevel(); got != 80 {
		t.Errorf("AdminLevel() = %d, want the default 80", got)
	}
}
