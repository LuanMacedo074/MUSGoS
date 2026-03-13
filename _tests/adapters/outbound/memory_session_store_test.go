package outbound_test

import (
	"testing"

	"fsos-server/internal/adapters/outbound"
)

func newMemoryStore() *outbound.MemorySessionStore {
	return outbound.NewMemorySessionStore()
}

func TestMemory_RegisterConnection(t *testing.T) {
	testRegisterConnection(t, newMemoryStore())
}

func TestMemory_IsConnected(t *testing.T) {
	testIsConnected(t, newMemoryStore())
}

func TestMemory_GetAllConnections(t *testing.T) {
	testGetAllConnections(t, newMemoryStore())
}

func TestMemory_UnregisterConnection_CleansEverything(t *testing.T) {
	testUnregisterConnection_CleansEverything(t, newMemoryStore())
}

func TestMemory_SessionAttribute_SetGetDelete(t *testing.T) {
	testSessionAttribute_SetGetDelete(t, newMemoryStore())
}

func TestMemory_SessionAttribute_Float(t *testing.T) {
	testSessionAttribute_Float(t, newMemoryStore())
}

func TestMemory_SessionAttribute_String(t *testing.T) {
	testSessionAttribute_String(t, newMemoryStore())
}

func TestMemory_SessionAttribute_Symbol(t *testing.T) {
	testSessionAttribute_Symbol(t, newMemoryStore())
}

func TestMemory_JoinRoom_And_GetMembers(t *testing.T) {
	testJoinRoom_And_GetMembers(t, newMemoryStore())
}

func TestMemory_LeaveRoom(t *testing.T) {
	testLeaveRoom(t, newMemoryStore())
}

func TestMemory_GetClientRooms(t *testing.T) {
	testGetClientRooms(t, newMemoryStore())
}

func TestMemory_LeaveAllRooms(t *testing.T) {
	testLeaveAllRooms(t, newMemoryStore())
}

func TestMemory_GetConnection_NonExistent(t *testing.T) {
	testGetConnection_NonExistent(t, newMemoryStore())
}
