package testutil

import (
	"sync"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// LogEntry represents a single log call captured by MockLogger.
type LogEntry struct {
	Level  ports.LogLevel
	Msg    string
	Fields map[string]interface{}
}

// MockLogger implements ports.Logger and records all calls for inspection.
type MockLogger struct {
	Messages []LogEntry
}

func (m *MockLogger) Debug(msg string, fields ...map[string]interface{}) {
	m.record(ports.DEBUG, msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...map[string]interface{}) {
	m.record(ports.INFO, msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...map[string]interface{}) {
	m.record(ports.WARN, msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...map[string]interface{}) {
	m.record(ports.ERROR, msg, fields)
}

func (m *MockLogger) Fatal(msg string, fields ...map[string]interface{}) {
	m.record(ports.FATAL, msg, fields)
}

func (m *MockLogger) record(level ports.LogLevel, msg string, fields []map[string]interface{}) {
	entry := LogEntry{Level: level, Msg: msg}
	if len(fields) > 0 {
		entry.Fields = fields[0]
	}
	m.Messages = append(m.Messages, entry)
}

// MockCipher implements ports.Cipher with configurable behavior.
type MockCipher struct {
	DecryptFunc  func([]byte) []byte
	EncryptFunc  func([]byte) []byte
	DecryptCalls int
	EncryptCalls int
}

func (m *MockCipher) Encrypt(data []byte) []byte {
	m.EncryptCalls++
	if m.EncryptFunc != nil {
		return m.EncryptFunc(data)
	}
	return data
}

func (m *MockCipher) Decrypt(data []byte) []byte {
	m.DecryptCalls++
	if m.DecryptFunc != nil {
		return m.DecryptFunc(data)
	}
	return data
}

// MockSessionStore implements ports.SessionStore with in-memory maps.
type MockSessionStore struct {
	mu          sync.RWMutex
	connections map[string]*ports.ConnectionInfo
	attributes  map[string]map[string]lingo.LValue // clientID -> attrName -> value
	rooms       map[string]map[string]bool          // roomName -> clientIDs
	clientRooms map[string]map[string]bool          // clientID -> roomNames
}

func NewMockSessionStore() *MockSessionStore {
	return &MockSessionStore{
		connections: make(map[string]*ports.ConnectionInfo),
		attributes:  make(map[string]map[string]lingo.LValue),
		rooms:       make(map[string]map[string]bool),
		clientRooms: make(map[string]map[string]bool),
	}
}

func (m *MockSessionStore) RegisterConnection(clientID, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[clientID] = &ports.ConnectionInfo{
		ClientID:    clientID,
		IP:          ip,
		ConnectedAt: time.Now(),
	}
	return nil
}

func (m *MockSessionStore) UnregisterConnection(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.connections, clientID)
	delete(m.attributes, clientID)

	if rooms, ok := m.clientRooms[clientID]; ok {
		for room := range rooms {
			if members, ok := m.rooms[room]; ok {
				delete(members, clientID)
			}
		}
		delete(m.clientRooms, clientID)
	}
	return nil
}

func (m *MockSessionStore) GetConnection(clientID string) (*ports.ConnectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[clientID], nil
}

func (m *MockSessionStore) GetAllConnections() ([]ports.ConnectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conns := make([]ports.ConnectionInfo, 0, len(m.connections))
	for _, c := range m.connections {
		conns = append(conns, *c)
	}
	return conns, nil
}

func (m *MockSessionStore) IsConnected(clientID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.connections[clientID]
	return ok, nil
}

func (m *MockSessionStore) SetUserAttribute(clientID, attrName string, value lingo.LValue) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.attributes[clientID] == nil {
		m.attributes[clientID] = make(map[string]lingo.LValue)
	}
	m.attributes[clientID][attrName] = value
	return nil
}

func (m *MockSessionStore) GetUserAttribute(clientID, attrName string) (lingo.LValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if attrs, ok := m.attributes[clientID]; ok {
		if v, ok := attrs[attrName]; ok {
			return v, nil
		}
	}
	return lingo.NewLVoid(), nil
}

func (m *MockSessionStore) GetUserAttributeNames(clientID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	if attrs, ok := m.attributes[clientID]; ok {
		for name := range attrs {
			names = append(names, name)
		}
	}
	return names, nil
}

func (m *MockSessionStore) DeleteUserAttribute(clientID, attrName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if attrs, ok := m.attributes[clientID]; ok {
		delete(attrs, attrName)
	}
	return nil
}

func (m *MockSessionStore) JoinRoom(roomName, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rooms[roomName] == nil {
		m.rooms[roomName] = make(map[string]bool)
	}
	m.rooms[roomName][clientID] = true
	if m.clientRooms[clientID] == nil {
		m.clientRooms[clientID] = make(map[string]bool)
	}
	m.clientRooms[clientID][roomName] = true
	return nil
}

func (m *MockSessionStore) LeaveRoom(roomName, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if members, ok := m.rooms[roomName]; ok {
		delete(members, clientID)
	}
	if rooms, ok := m.clientRooms[clientID]; ok {
		delete(rooms, roomName)
	}
	return nil
}

func (m *MockSessionStore) GetRoomMembers(roomName string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var members []string
	if m.rooms[roomName] != nil {
		for id := range m.rooms[roomName] {
			members = append(members, id)
		}
	}
	return members, nil
}

func (m *MockSessionStore) GetClientRooms(clientID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var rooms []string
	if m.clientRooms[clientID] != nil {
		for room := range m.clientRooms[clientID] {
			rooms = append(rooms, room)
		}
	}
	return rooms, nil
}

func (m *MockSessionStore) LeaveAllRooms(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rooms, ok := m.clientRooms[clientID]; ok {
		for room := range rooms {
			if members, ok := m.rooms[room]; ok {
				delete(members, clientID)
			}
		}
		delete(m.clientRooms, clientID)
	}
	return nil
}

func (m *MockSessionStore) Close() error {
	return nil
}
