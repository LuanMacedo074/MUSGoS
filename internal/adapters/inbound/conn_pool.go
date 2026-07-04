package inbound

import (
	"fmt"
	"net"
	"sync"
)

type ConnPool struct {
	mu       sync.Mutex
	clients  map[string]net.Conn
	connToID map[net.Conn]string
	writeMu  map[net.Conn]*sync.Mutex
}

func NewConnPool() *ConnPool {
	return &ConnPool{
		clients:  make(map[string]net.Conn),
		connToID: make(map[net.Conn]string),
		writeMu:  make(map[net.Conn]*sync.Mutex),
	}
}

func (p *ConnPool) Register(conn net.Conn, clientID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[clientID] = conn
	p.connToID[conn] = clientID
	p.writeMu[conn] = &sync.Mutex{}
}

func (p *ConnPool) Unregister(conn net.Conn) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	clientID := p.connToID[conn]
	delete(p.connToID, conn)
	delete(p.clients, clientID)
	delete(p.writeMu, conn)
	return clientID
}

func (p *ConnPool) CurrentID(conn net.Conn) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connToID[conn]
}

func (p *ConnPool) WriteToClient(clientID string, data []byte) error {
	p.mu.Lock()
	conn, ok := p.clients[clientID]
	if !ok {
		p.mu.Unlock()
		return fmt.Errorf("client %q not connected", clientID)
	}
	wmu := p.writeMu[conn]
	p.mu.Unlock()

	wmu.Lock()
	defer wmu.Unlock()
	_, err := conn.Write(data)
	return err
}

// RemapClientID rebinds a connection from oldID to newID (used at Logon to swap
// the transient connection id for the authenticated userID). It returns false
// without changing anything if oldID is unknown, or if newID is already bound to
// a *different* connection — refusing to clobber another client's mapping guards
// against duplicate-userID session hijack. (backlog H3)
func (p *ConnPool) RemapClientID(oldID, newID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	conn, ok := p.clients[oldID]
	if !ok {
		return false
	}
	if existing, taken := p.clients[newID]; taken && existing != conn {
		return false
	}
	delete(p.clients, oldID)
	p.clients[newID] = conn
	p.connToID[conn] = newID
	return true
}

func (p *ConnPool) DisconnectClient(clientID string) error {
	p.mu.Lock()
	conn, ok := p.clients[clientID]
	p.mu.Unlock()
	if !ok {
		return fmt.Errorf("client %q not connected", clientID)
	}
	return conn.Close()
}

func (p *ConnPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for conn := range p.connToID {
		conn.Close()
	}
}
