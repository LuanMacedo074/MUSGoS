package ports

import (
	"time"

	"fsos-server/internal/domain/types/lingo"
)

type ConnectionInfo struct {
	ClientID       string
	IP             string
	ConnectedAt    time.Time
	LastActivityAt time.Time
}

type SessionStore interface {
	// Connection lifecycle
	RegisterConnection(clientID, ip string) error
	UnregisterConnection(clientID string) error
	GetConnection(clientID string) (*ConnectionInfo, error)
	GetAllConnections() ([]ConnectionInfo, error)
	IsConnected(clientID string) (bool, error)
	UpdateLastActivity(clientID string) error

	// Session attributes (ephemeral per clientID)
	SetUserAttribute(clientID, attrName string, value lingo.LValue) error
	GetUserAttribute(clientID, attrName string) (lingo.LValue, error)
	GetUserAttributeNames(clientID string) ([]string, error)
	DeleteUserAttribute(clientID, attrName string) error

	// Room management
	JoinRoom(roomName, clientID string) error
	LeaveRoom(roomName, clientID string) error
	GetRoomMembers(roomName string) ([]string, error)
	GetClientRooms(clientID string) ([]string, error)
	LeaveAllRooms(clientID string) error

	Close() error
}
