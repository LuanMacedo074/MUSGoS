package services

import (
	"fsos-server/internal/domain/ports"
)

// defaultAdminLevel is the cross-user privilege threshold when no
// DBAdmin.createUser level is configured.
const defaultAdminLevel = 80

// Authorizer owns command permission policy: whether a sender's session
// level clears a command's configured level (deny-by-default for unlisted
// commands), and whether a sender may act on another user's data. It reads
// the user level LogonService stamped into the session.
type Authorizer struct {
	sessions      ports.SessionStore
	commandLevels map[string]int
}

func NewAuthorizer(sessions ports.SessionStore, commandLevels map[string]int) *Authorizer {
	return &Authorizer{sessions: sessions, commandLevels: commandLevels}
}

// UserLevel reports the sender's session user level, 0 when absent.
func (a *Authorizer) UserLevel(userID string) int {
	val, err := a.sessions.GetUserAttribute(userID, UserLevelAttribute)
	if err != nil {
		return 0
	}
	return int(val.ToInteger())
}

// CanRun reports whether senderID may run command. Commands without a
// configured level are denied for everyone (deny-by-default).
func (a *Authorizer) CanRun(senderID, command string) bool {
	requiredLevel, ok := a.commandLevels[command]
	if !ok {
		return false
	}
	return a.UserLevel(senderID) >= requiredLevel
}

// AdminLevel is the level required to act on data the caller does not own. It
// tracks the configured DBAdmin level (default 80) so cross-user player-data
// access needs the same privilege as user administration.
func (a *Authorizer) AdminLevel() int {
	if lvl, ok := a.commandLevels["DBAdmin.createUser"]; ok {
		return lvl
	}
	return defaultAdminLevel
}

// OwnerOrAdmin reports whether senderID may operate on targetUserID's data:
// only when it is their own data, or they hold an admin-level session. (H2)
func (a *Authorizer) OwnerOrAdmin(senderID, targetUserID string) bool {
	return targetUserID == senderID || a.UserLevel(senderID) >= a.AdminLevel()
}
