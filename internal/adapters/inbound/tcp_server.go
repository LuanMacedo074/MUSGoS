package inbound

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"

	"fsos-server/internal/domain/ports"
)

type TCPServerConfig struct {
	Port           string
	ServerIP       string
	MaxMessageSize int
	TCPNoDelay     bool
}

type TCPServer struct {
	config         TCPServerConfig
	listener       net.Listener
	logger         ports.Logger
	shutdown       chan bool
	wg             sync.WaitGroup
	messageHandler ports.MessageHandler
	sessionStore   ports.SessionStore
	mu             sync.Mutex
	conns          map[net.Conn]struct{}
}

func NewTCPServer(cfg TCPServerConfig, logger ports.Logger, handler ports.MessageHandler, sessionStore ports.SessionStore) *TCPServer {
	return &TCPServer{
		config:         cfg,
		logger:         logger,
		shutdown:       make(chan bool),
		messageHandler: handler,
		sessionStore:   sessionStore,
		conns:          make(map[net.Conn]struct{}),
	}
}

func (s *TCPServer) Start(ready chan struct{}) error {
	addr := s.config.ServerIP + ":" + s.config.Port
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.logger.Info("TCP Server listening", map[string]interface{}{
		"address": addr,
	})

	if ready != nil {
		close(ready)
	}

	for {
		select {
		case <-s.shutdown:
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.shutdown:
					return nil
				default:
					s.logger.Error("Failed to accept connection", map[string]interface{}{
						"error": err,
					})
					continue
				}
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	s.mu.Lock()
	s.conns[conn] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	clientIP := conn.RemoteAddr().String()

	if err := s.sessionStore.RegisterConnection(clientIP, clientIP); err != nil {
		s.logger.Error("Failed to register connection", map[string]interface{}{
			"client": clientIP,
			"error":  err.Error(),
		})
	}

	defer func() {
		if err := s.sessionStore.UnregisterConnection(clientIP); err != nil {
			s.logger.Error("Failed to unregister connection", map[string]interface{}{
				"client": clientIP,
				"error":  err.Error(),
			})
		}
		conn.Close()
	}()

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(s.config.TCPNoDelay)
	}

	s.logger.Info("New connection established", map[string]interface{}{
		"client": clientIP,
	})

	reader := bufio.NewReader(conn)
	buffer := make([]byte, s.config.MaxMessageSize)
	totalBytes := 0

	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				s.logger.Error("Read error", map[string]interface{}{
					"client": clientIP,
					"error":  err,
				})
			}
			break
		}

		if n > 0 {
			s.logger.Info("TCP Packet Capture", map[string]interface{}{
				"client": clientIP,
				"offset": fmt.Sprintf("0x%04X", totalBytes),
				"bytes":  n,
			})

			s.logger.Debug("Processing message", map[string]interface{}{
				"client": clientIP,
				"bytes":  n,
			})

			response, err := s.messageHandler.HandleRawMessage(clientIP, buffer[:n])
			if err != nil {
				s.logger.Error("Message handler error", map[string]interface{}{
					"client": clientIP,
					"error":  err.Error(),
				})
			}

			if len(response) > 0 {
				_, writeErr := conn.Write(response)
				if writeErr != nil {
					s.logger.Error("Failed to send response", map[string]interface{}{
						"client": clientIP,
						"error":  writeErr.Error(),
					})
					break
				}
			}

			totalBytes += n
		}
	}

	s.logger.Info("Connection closed", map[string]interface{}{
		"client":      clientIP,
		"total_bytes": totalBytes,
	})
}

func (s *TCPServer) Shutdown() {
	s.logger.Info("Shutting down TCP server...")
	close(s.shutdown)

	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("TCP server shutdown complete")
}
