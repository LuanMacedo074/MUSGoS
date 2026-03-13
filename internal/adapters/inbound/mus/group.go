package mus

import (
	"fmt"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
	"strings"
	"sync"
)

type Group struct {
	Name       string
	movieID    string
	persistent bool // if true, don't delete when empty (e.g., @AllUsers)
	mu         sync.RWMutex
	attributes map[string]lingo.LValue
}

func NewGroup(name, movieID string, persistent bool) *Group {
	return &Group{
		Name:       name,
		movieID:    movieID,
		persistent: persistent,
		attributes: make(map[string]lingo.LValue),
	}
}

func (g *Group) GetAttribute(name string) lingo.LValue {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if v, ok := g.attributes[name]; ok {
		return v
	}
	return lingo.NewLVoid()
}

func (g *Group) SetAttribute(name string, value lingo.LValue) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.attributes[name] = value
}

func (g *Group) DeleteAttribute(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.attributes, name)
}

func (g *Group) GetAttributeNames() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	names := make([]string, 0, len(g.attributes))
	for name := range g.attributes {
		names = append(names, name)
	}
	return names
}

func groupRoomName(movieID, groupName string) string {
	return fmt.Sprintf("%s:%s", movieID, groupName)
}

type GroupManager struct {
	sessionStore ports.SessionStore
	logger       ports.Logger
}

func NewGroupManager(sessionStore ports.SessionStore, logger ports.Logger) *GroupManager {
	return &GroupManager{
		sessionStore: sessionStore,
		logger:       logger,
	}
}

func (gm *GroupManager) JoinGroup(movieID, groupName, userID string) error {
	// Verify user is in the movie
	members, err := gm.sessionStore.GetRoomMembers(movieRoomName(movieID))
	if err != nil {
		return fmt.Errorf("failed to check movie membership: %w", err)
	}

	inMovie := false
	for _, m := range members {
		if m == userID {
			inMovie = true
			break
		}
	}
	if !inMovie {
		return fmt.Errorf("user %q is not in movie %q", userID, movieID)
	}

	roomName := groupRoomName(movieID, groupName)
	if err := gm.sessionStore.JoinRoom(roomName, userID); err != nil {
		return fmt.Errorf("failed to join group room: %w", err)
	}

	gm.logger.Info("User joined group", map[string]interface{}{
		"movieID":   movieID,
		"groupName": groupName,
		"userID":    userID,
	})

	return nil
}

func (gm *GroupManager) LeaveGroup(movieID, groupName, userID string) error {
	roomName := groupRoomName(movieID, groupName)
	if err := gm.sessionStore.LeaveRoom(roomName, userID); err != nil {
		return fmt.Errorf("failed to leave group room: %w", err)
	}

	gm.logger.Info("User left group", map[string]interface{}{
		"movieID":   movieID,
		"groupName": groupName,
		"userID":    userID,
	})

	return nil
}

func (gm *GroupManager) GetGroupMembers(movieID, groupName string) ([]string, error) {
	return gm.sessionStore.GetRoomMembers(groupRoomName(movieID, groupName))
}

func (gm *GroupManager) GetGroupMemberCount(movieID, groupName string) (int, error) {
	members, err := gm.sessionStore.GetRoomMembers(groupRoomName(movieID, groupName))
	if err != nil {
		return 0, err
	}
	return len(members), nil
}

func (gm *GroupManager) LeaveAllGroups(movieID, userID string) error {
	rooms, err := gm.sessionStore.GetClientRooms(userID)
	if err != nil {
		return fmt.Errorf("failed to get client rooms: %w", err)
	}

	prefix := movieID + ":"
	for _, room := range rooms {
		if strings.HasPrefix(room, prefix) {
			if err := gm.sessionStore.LeaveRoom(room, userID); err != nil {
				return fmt.Errorf("failed to leave group room %q: %w", room, err)
			}
		}
	}

	return nil
}
