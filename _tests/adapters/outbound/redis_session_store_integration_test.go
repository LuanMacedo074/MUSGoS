//go:build integration

package outbound_test

import (
	"testing"

	"fsos-server/internal/domain/ports"
)

// The Redis session store runs the same behavioral helpers as the in-memory
// store (session_store_test.go), against a real Redis. Each subtest gets a fresh
// store with a unique key prefix (via newRedisSessionStore -> safeName), so they
// don't collide on the shared server.
func TestIntegrationRedisSessionStore(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*testing.T, ports.SessionStore)
	}{
		{"RegisterConnection", testRegisterConnection},
		{"IsConnected", testIsConnected},
		{"GetAllConnections", testGetAllConnections},
		{"UnregisterConnection_CleansEverything", testUnregisterConnection_CleansEverything},
		{"SessionAttribute_SetGetDelete", testSessionAttribute_SetGetDelete},
		{"SessionAttribute_Float", testSessionAttribute_Float},
		{"SessionAttribute_String", testSessionAttribute_String},
		{"SessionAttribute_Symbol", testSessionAttribute_Symbol},
		{"JoinRoom_And_GetMembers", testJoinRoom_And_GetMembers},
		{"LeaveRoom", testLeaveRoom},
		{"GetClientRooms", testGetClientRooms},
		{"LeaveAllRooms", testLeaveAllRooms},
		{"GetConnection_NonExistent", testGetConnection_NonExistent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t, newRedisSessionStore(t))
		})
	}
}
