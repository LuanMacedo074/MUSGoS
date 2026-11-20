package inbound_test

import (
	"encoding/binary"
	"io"
	"runtime"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func TestIdleDisconnect_PreservesSessionUntilHookAndRunsOnce(t *testing.T) {
	for _, mode := range []string{"idle", "socket", "race"} {
		t.Run(mode, func(t *testing.T) {
			store := outbound.NewMemorySessionStore()
			pool := inbound.NewConnPool()
			logger := &testutil.MockLogger{}
			hooks := make(chan bool, 4)
			client, stop := testutil.TCPClient(t, inbound.TCPServerDeps{Pool: pool, SessionStore: store, OnDisconnect: func(id string) {
				value, _ := store.GetUserAttribute(id, "state")
				rooms, _ := store.GetClientRooms(id)
				hooks <- lingo.StringValue(value) == "present" && len(rooms) == 1
			}})
			id := client.LocalAddr().String()
			store.SetUserAttribute(id, "state", lingo.NewLString("present"))
			store.JoinRoom("room", id)
			checker := inbound.NewIdleChecker(store, pool, logger, 30*time.Second)
			conn, _ := store.GetConnection(id)
			now := conn.LastActivityAt.Add(31 * time.Second)
			switch mode {
			case "idle":
				checker.Check(now)
			case "socket":
				client.Close()
			case "race":
				closed := make(chan struct{})
				go func() { client.Close(); close(closed) }()
				checker.Check(now)
				<-closed
			}
			select {
			case intact := <-hooks:
				if !intact {
					t.Error("hook lost session attributes or rooms")
				}
			case <-time.After(time.Second):
				t.Fatal("disconnect hook did not run")
			}
			// Shutdown joins the handler, making hook count and session removal observable.
			// A second idle sweep must never invoke the hook itself.
			checker.Check(now)
			stop()
			select {
			case <-hooks:
				t.Error("disconnect hook ran more than once")
			default:
			}
			connected, _ := store.IsConnected(id)
			if connected {
				t.Error("session survived teardown")
			}

		})
	}
}

func TestIdleDisconnect_OnlyPingsKeepConnectionAlive(t *testing.T) {
	store := outbound.NewMemorySessionStore()
	pool := inbound.NewConnPool()
	logger := &testutil.MockLogger{}
	sender := mus.NewSender(pool, store, logger, nil, false, "test")
	// A keepalive script that answers each ping, as an application would.
	scripts := testutil.NewScriptsDir(t)
	testutil.WriteScriptNested(t, scripts, "misc", "ping", `mus.sendMessage(mus.getSender(), "pong", mus.getSender())`)
	engine := outbound.NewLuaScriptEngine(scripts, logger, 1, nil, sender, nil, nil, store, nil, nil)
	dispatcher := mus.NewDispatcher(logger, engine, nil, sender, nil)
	handler := inbound.NewSMUSHandler(logger, nil, dispatcher, false)
	client, stop := testutil.TCPClient(t, inbound.TCPServerDeps{Pool: pool, SessionStore: store, Handler: handler})
	id := client.LocalAddr().String()
	checker := inbound.NewIdleChecker(store, pool, logger, 30*time.Second)
	for i := 0; i < 5; i++ {
		before, _ := store.GetConnection(id)
		if before == nil {
			t.Fatal("pinging connection was dropped")
		}
		if _, err := client.Write(buildValidSMUSMessage("misc/ping", id, []string{"system.script"})); err != nil {
			t.Fatal(err)
		}
		client.SetReadDeadline(time.Now().Add(time.Second))
		// The script sends its pong, then the dispatcher sends its void result.
		for _, subject := range []string{"pong", "misc/ping"} {
			header := make([]byte, 6)
			if _, err := io.ReadFull(client, header); err != nil {
				t.Fatal(err)
			}
			payload := make([]byte, binary.BigEndian.Uint32(header[2:6]))
			if _, err := io.ReadFull(client, payload); err != nil {
				t.Fatal(err)
			}
			reply, err := smus.ParseMUSMessageWithDecryption(append(header, payload...), nil)
			if err != nil {
				t.Fatal(err)
			}
			if reply.Subject.Value != subject {
				t.Fatalf("reply subject = %q, want %q", reply.Subject.Value, subject)
			}
		}

		deadline := time.Now().Add(time.Second)
		for {
			after, _ := store.GetConnection(id)
			if after != nil && after.LastActivityAt.After(before.LastActivityAt) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("misc/ping did not refresh LastActivityAt")
			}
			runtime.Gosched()
		}
		// The old activity would now be expired; the ping must keep it alive.
		checker.Check(before.LastActivityAt.Add(30*time.Second + time.Nanosecond))
	}
	connected, _ := store.IsConnected(id)
	if !connected {
		t.Fatal("ping-only client was disconnected")
	}
	stop()
}

func TestTCPActivity_RefreshesEvenWhenDispatchRejectsMessage(t *testing.T) {
	store := outbound.NewMemorySessionStore()
	pool := inbound.NewConnPool()
	logger := &testutil.MockLogger{}
	handler := inbound.NewSMUSHandler(logger, nil, mus.NewDispatcher(logger, nil, nil, nil, nil), false)
	client, stop := testutil.TCPClient(t, inbound.TCPServerDeps{Pool: pool, SessionStore: store, Handler: handler})
	id := client.LocalAddr().String()
	before, _ := store.GetConnection(id)
	// A valid frame with no recipients is rejected by the dispatcher.
	if _, err := client.Write(buildValidSMUSMessage("misc/ping", id, nil)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		after, _ := store.GetConnection(id)
		if after != nil && after.LastActivityAt.After(before.LastActivityAt) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("rejected inbound message did not refresh activity")
		}
		runtime.Gosched()
	}
	stop()
}
