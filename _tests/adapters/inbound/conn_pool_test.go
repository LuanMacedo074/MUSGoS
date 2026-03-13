package inbound_test

import (
	"net"
	"sync"
	"testing"

	"fsos-server/internal/adapters/inbound"
)

func TestConnPool_WriteToClient_Success(t *testing.T) {
	pool := inbound.NewConnPool()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	pool.Register(server, "user1")

	data := []byte("hello")
	go func() {
		pool.WriteToClient("user1", data)
	}()

	buf := make([]byte, 64)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("got %q, want %q", string(buf[:n]), "hello")
	}
}

func TestConnPool_WriteToClient_UnknownClient(t *testing.T) {
	pool := inbound.NewConnPool()

	err := pool.WriteToClient("nobody", []byte("data"))
	if err == nil {
		t.Error("expected error for unknown client")
	}
}

func TestConnPool_RemapClientID(t *testing.T) {
	pool := inbound.NewConnPool()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	pool.Register(server, "ip:1234")
	pool.RemapClientID("ip:1234", "user1")

	// Old ID should fail
	err := pool.WriteToClient("ip:1234", []byte("data"))
	if err == nil {
		t.Error("expected error for old client ID after remap")
	}

	// New ID should work
	go func() {
		pool.WriteToClient("user1", []byte("hi"))
	}()

	buf := make([]byte, 64)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(buf[:n]) != "hi" {
		t.Errorf("got %q, want %q", string(buf[:n]), "hi")
	}
}

func TestConnPool_ConcurrentWrites(t *testing.T) {
	pool := inbound.NewConnPool()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	pool.Register(server, "user1")

	const numWriters = 10
	payload := []byte("x")

	var wg sync.WaitGroup
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()
			pool.WriteToClient("user1", payload)
		}()
	}

	// Read all bytes
	received := 0
	buf := make([]byte, 64)
	for received < numWriters {
		n, err := client.Read(buf)
		if err != nil {
			t.Fatalf("read error after %d bytes: %v", received, err)
		}
		received += n
	}

	go func() {
		wg.Wait()
	}()

	if received != numWriters {
		t.Errorf("received %d bytes, want %d", received, numWriters)
	}
}

func TestConnPool_WriteAfterRemap(t *testing.T) {
	pool := inbound.NewConnPool()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	pool.Register(server, "old")
	pool.RemapClientID("old", "new")

	// CurrentID should return the new ID
	if id := pool.CurrentID(server); id != "new" {
		t.Errorf("CurrentID after remap = %q, want %q", id, "new")
	}

	// WriteToClient with new ID should succeed
	go func() {
		pool.WriteToClient("new", []byte("after-remap"))
	}()

	buf := make([]byte, 64)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(buf[:n]) != "after-remap" {
		t.Errorf("got %q, want %q", string(buf[:n]), "after-remap")
	}
}

func TestConnPool_Unregister(t *testing.T) {
	pool := inbound.NewConnPool()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	pool.Register(server, "user1")
	returnedID := pool.Unregister(server)

	if returnedID != "user1" {
		t.Errorf("Unregister returned %q, want %q", returnedID, "user1")
	}

	err := pool.WriteToClient("user1", []byte("data"))
	if err == nil {
		t.Error("expected error after unregister")
	}
}
