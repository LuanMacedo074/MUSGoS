package outbound_test

import (
	"testing"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

func testRegisterConnection(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.RegisterConnection("client1", "192.168.1.1:5000"))

	conn, err := store.GetConnection("client1")
	mustNoErr(t, err)
	if conn == nil {
		t.Fatal("expected connection info, got nil")
	}
	if conn.IP != "192.168.1.1:5000" {
		t.Errorf("expected IP '192.168.1.1:5000', got %q", conn.IP)
	}
	if conn.ClientID != "client1" {
		t.Errorf("expected clientID 'client1', got %q", conn.ClientID)
	}
}

func testIsConnected(t *testing.T, store ports.SessionStore) {
	t.Helper()

	connected, err := store.IsConnected("client1")
	mustNoErr(t, err)
	if connected {
		t.Error("expected not connected before register")
	}

	mustNoErr(t, store.RegisterConnection("client1", "192.168.1.1:5000"))

	connected, err = store.IsConnected("client1")
	mustNoErr(t, err)
	if !connected {
		t.Error("expected connected after register")
	}
}

func testGetAllConnections(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.RegisterConnection("client1", "192.168.1.1:5000"))
	mustNoErr(t, store.RegisterConnection("client2", "192.168.1.2:5000"))

	conns, err := store.GetAllConnections()
	mustNoErr(t, err)
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

func testUnregisterConnection_CleansEverything(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.RegisterConnection("client1", "192.168.1.1:5000"))
	mustNoErr(t, store.SetUserAttribute("client1", "token", lingo.NewLString("abc")))
	mustNoErr(t, store.JoinRoom("lobby", "client1"))

	mustNoErr(t, store.UnregisterConnection("client1"))

	connected, err := store.IsConnected("client1")
	mustNoErr(t, err)
	if connected {
		t.Error("expected not connected after unregister")
	}

	got, err := store.GetUserAttribute("client1", "token")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtVoid {
		t.Error("expected void for attribute after unregister")
	}

	members, err := store.GetRoomMembers("lobby")
	mustNoErr(t, err)
	if len(members) != 0 {
		t.Errorf("expected 0 room members after unregister, got %d", len(members))
	}
}

func testSessionAttribute_SetGetDelete(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.SetUserAttribute("client1", "score", lingo.NewLInteger(42)))

	got, err := store.GetUserAttribute("client1", "score")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtInteger {
		t.Errorf("expected integer type, got %d", got.GetType())
	}
	if got.ToInteger() != 42 {
		t.Errorf("expected 42, got %d", got.ToInteger())
	}

	got2, err := store.GetUserAttribute("client2", "score")
	mustNoErr(t, err)
	if got2.GetType() != lingo.VtVoid {
		t.Error("different client should not have the attribute")
	}

	mustNoErr(t, store.SetUserAttribute("client1", "level", lingo.NewLInteger(5)))
	names, err := store.GetUserAttributeNames("client1")
	mustNoErr(t, err)
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}

	mustNoErr(t, store.DeleteUserAttribute("client1", "score"))
	got3, err := store.GetUserAttribute("client1", "score")
	mustNoErr(t, err)
	if got3.GetType() != lingo.VtVoid {
		t.Error("expected void after delete")
	}
}

func testSessionAttribute_Float(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.SetUserAttribute("client1", "ratio", lingo.NewLFloat(3.14)))

	got, err := store.GetUserAttribute("client1", "ratio")
	mustNoErr(t, err)
	if got.GetType() != lingo.VtFloat {
		t.Errorf("expected float type, got %d", got.GetType())
	}
	if got.ToDouble() != 3.14 {
		t.Errorf("expected 3.14, got %f", got.ToDouble())
	}
}

func testSessionAttribute_String(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.SetUserAttribute("client1", "name", lingo.NewLString("hello")))

	got, err := store.GetUserAttribute("client1", "name")
	mustNoErr(t, err)
	ls, ok := got.(*lingo.LString)
	if !ok {
		t.Fatalf("expected *LString, got %T", got)
	}
	if ls.Value != "hello" {
		t.Errorf("expected 'hello', got %q", ls.Value)
	}
}

func testSessionAttribute_Symbol(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.SetUserAttribute("client1", "sym", lingo.NewLSymbol("mySymbol")))

	got, err := store.GetUserAttribute("client1", "sym")
	mustNoErr(t, err)
	ls, ok := got.(*lingo.LSymbol)
	if !ok {
		t.Fatalf("expected *LSymbol, got %T", got)
	}
	if ls.Value != "mySymbol" {
		t.Errorf("expected 'mySymbol', got %q", ls.Value)
	}
}

func testJoinRoom_And_GetMembers(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.JoinRoom("lobby", "client1"))
	mustNoErr(t, store.JoinRoom("lobby", "client2"))

	members, err := store.GetRoomMembers("lobby")
	mustNoErr(t, err)
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func testLeaveRoom(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.JoinRoom("lobby", "client1"))
	mustNoErr(t, store.JoinRoom("lobby", "client2"))
	mustNoErr(t, store.LeaveRoom("lobby", "client1"))

	members, err := store.GetRoomMembers("lobby")
	mustNoErr(t, err)
	if len(members) != 1 {
		t.Errorf("expected 1 member after leave, got %d", len(members))
	}
}

func testGetClientRooms(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.JoinRoom("lobby", "client1"))
	mustNoErr(t, store.JoinRoom("game", "client1"))

	rooms, err := store.GetClientRooms("client1")
	mustNoErr(t, err)
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}
}

func testLeaveAllRooms(t *testing.T, store ports.SessionStore) {
	t.Helper()

	mustNoErr(t, store.JoinRoom("lobby", "client1"))
	mustNoErr(t, store.JoinRoom("game", "client1"))
	mustNoErr(t, store.LeaveAllRooms("client1"))

	rooms, err := store.GetClientRooms("client1")
	mustNoErr(t, err)
	if len(rooms) != 0 {
		t.Errorf("expected 0 rooms after leave all, got %d", len(rooms))
	}

	members, err := store.GetRoomMembers("lobby")
	mustNoErr(t, err)
	if len(members) != 0 {
		t.Errorf("expected 0 members in lobby, got %d", len(members))
	}
}

func testGetConnection_NonExistent(t *testing.T, store ports.SessionStore) {
	t.Helper()

	conn, err := store.GetConnection("nonexistent")
	mustNoErr(t, err)
	if conn != nil {
		t.Error("expected nil for non-existent connection")
	}
}
