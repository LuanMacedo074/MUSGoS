package testutil

import (
	"net"
	"sync"
	"testing"
	"time"

	"fsos-server/internal/adapters/inbound"
)

// TCPClient starts the real TCP server on an ephemeral loopback port and waits
// until its client is registered. Cleanup waits for all disconnect hooks.
func TCPClient(t *testing.T, deps inbound.TCPServerDeps) (net.Conn, func()) {
	t.Helper()
	logger := &tcpLogger{address: make(chan string, 1), connected: make(chan struct{}, 1)}
	deps.Logger = logger
	server := inbound.NewTCPServer(inbound.TCPServerConfig{ServerIP: "127.0.0.1", Port: "0", MaxMessageSize: 4096}, deps)
	stopped := make(chan error, 1)
	go func() { stopped <- server.Start(nil) }()
	var addr string
	select {
	case addr = <-logger.address:
	case err := <-stopped:
		t.Fatalf("start TCP server: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("TCP server did not start")
	}
	var once sync.Once
	stop := func() {
		once.Do(func() {
			server.Shutdown()
			if err := <-stopped; err != nil {
				t.Error(err)
			}
		})
	}
	t.Cleanup(stop)
	client, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	select {
	case <-logger.connected:
	case <-time.After(3 * time.Second):
		t.Fatal("client was not registered")
	}
	return client, stop
}

type tcpLogger struct {
	address   chan string
	connected chan struct{}
}

func (*tcpLogger) Debug(string, ...map[string]interface{}) {}
func (l *tcpLogger) Info(msg string, fields ...map[string]interface{}) {
	if msg == "TCP Server listening" {
		l.address <- fields[0]["address"].(string)
	}
	if msg == "New connection established" {
		l.connected <- struct{}{}
	}
}
func (*tcpLogger) Warn(string, ...map[string]interface{})  {}
func (*tcpLogger) Error(string, ...map[string]interface{}) {}
func (*tcpLogger) Fatal(string, ...map[string]interface{}) {}
func (*tcpLogger) Flush()                                  {}
