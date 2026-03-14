package outbound

import (
	"sync"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

type MemorySessionStore struct {
	mu          sync.RWMutex
	connections map[string]ports.ConnectionInfo
	attributes  map[string]map[string]lingo.LValue
	rooms       map[string]map[string]struct{}
	clientRooms map[string]map[string]struct{}
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		connections: make(map[string]ports.ConnectionInfo),
		attributes:  make(map[string]map[string]lingo.LValue),
		rooms:       make(map[string]map[string]struct{}),
		clientRooms: make(map[string]map[string]struct{}),
	}
}

// --- Connection lifecycle ---

func (m *MemorySessionStore) RegisterConnection(clientID, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.connections[clientID] = ports.ConnectionInfo{
		ClientID:       clientID,
		IP:             ip,
		ConnectedAt:    now,
		LastActivityAt: now,
	}
	return nil
}

func (m *MemorySessionStore) UnregisterConnection(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.connections, clientID)
	delete(m.attributes, clientID)

	if rooms, ok := m.clientRooms[clientID]; ok {
		for room := range rooms {
			if members, exists := m.rooms[room]; exists {
				delete(members, clientID)
				if len(members) == 0 {
					delete(m.rooms, room)
				}
			}
		}
		delete(m.clientRooms, clientID)
	}

	return nil
}

func (m *MemorySessionStore) GetConnection(clientID string) (*ports.ConnectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[clientID]
	if !ok {
		return nil, nil
	}
	return &conn, nil
}

func (m *MemorySessionStore) GetAllConnections() ([]ports.ConnectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.connections) == 0 {
		return []ports.ConnectionInfo{}, nil
	}

	conns := make([]ports.ConnectionInfo, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	return conns, nil
}

func (m *MemorySessionStore) UpdateLastActivity(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.connections[clientID]; ok {
		conn.LastActivityAt = time.Now().UTC()
		m.connections[clientID] = conn
	}
	return nil
}

func (m *MemorySessionStore) IsConnected(clientID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.connections[clientID]
	return ok, nil
}

// --- Session attributes ---

func (m *MemorySessionStore) SetUserAttribute(clientID, attrName string, value lingo.LValue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.attributes[clientID] == nil {
		m.attributes[clientID] = make(map[string]lingo.LValue)
	}
	m.attributes[clientID][attrName] = value
	return nil
}

func (m *MemorySessionStore) GetUserAttribute(clientID, attrName string) (lingo.LValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	attrs, ok := m.attributes[clientID]
	if !ok {
		return lingo.NewLVoid(), nil
	}
	val, ok := attrs[attrName]
	if !ok {
		return lingo.NewLVoid(), nil
	}
	return val, nil
}

func (m *MemorySessionStore) GetUserAttributeNames(clientID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	attrs, ok := m.attributes[clientID]
	if !ok {
		return nil, nil
	}
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	return names, nil
}

func (m *MemorySessionStore) DeleteUserAttribute(clientID, attrName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if attrs, ok := m.attributes[clientID]; ok {
		delete(attrs, attrName)
	}
	return nil
}

// --- Room management ---

func (m *MemorySessionStore) JoinRoom(roomName, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.rooms[roomName] == nil {
		m.rooms[roomName] = make(map[string]struct{})
	}
	m.rooms[roomName][clientID] = struct{}{}

	if m.clientRooms[clientID] == nil {
		m.clientRooms[clientID] = make(map[string]struct{})
	}
	m.clientRooms[clientID][roomName] = struct{}{}

	return nil
}

func (m *MemorySessionStore) LeaveRoom(roomName, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if members, ok := m.rooms[roomName]; ok {
		delete(members, clientID)
		if len(members) == 0 {
			delete(m.rooms, roomName)
		}
	}
	if rooms, ok := m.clientRooms[clientID]; ok {
		delete(rooms, roomName)
	}

	return nil
}

func (m *MemorySessionStore) GetRoomMembers(roomName string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members, ok := m.rooms[roomName]
	if !ok {
		return nil, nil
	}
	result := make([]string, 0, len(members))
	for id := range members {
		result = append(result, id)
	}
	return result, nil
}

func (m *MemorySessionStore) GetClientRooms(clientID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rooms, ok := m.clientRooms[clientID]
	if !ok {
		return nil, nil
	}
	result := make([]string, 0, len(rooms))
	for name := range rooms {
		result = append(result, name)
	}
	return result, nil
}

func (m *MemorySessionStore) LeaveAllRooms(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rooms, ok := m.clientRooms[clientID]; ok {
		for room := range rooms {
			if members, exists := m.rooms[room]; exists {
				delete(members, clientID)
				if len(members) == 0 {
					delete(m.rooms, room)
				}
			}
		}
		delete(m.clientRooms, clientID)
	}

	return nil
}

// --- Close ---

func (m *MemorySessionStore) Close() error {
	return nil
}
