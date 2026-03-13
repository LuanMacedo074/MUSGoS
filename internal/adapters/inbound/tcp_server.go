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
	pool           *ConnPool
}

func NewTCPServer(cfg TCPServerConfig, handler ports.MessageHandler, pool *ConnPool, logger ports.Logger, sessionStore ports.SessionStore) *TCPServer {
	return &TCPServer{
		config:         cfg,
		messageHandler: handler,
		pool:           pool,
		logger:         logger,
		shutdown:       make(chan bool),
		sessionStore:   sessionStore,
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

	clientIP := conn.RemoteAddr().String()

	s.pool.Register(conn, clientIP)

	defer func() {
		currentID := s.pool.Unregister(conn)

		if err := s.sessionStore.UnregisterConnection(currentID); err != nil {
			s.logger.Error("Failed to unregister connection", map[string]interface{}{
				"client": currentID,
				"error":  err.Error(),
			})
		}
		conn.Close()
	}()

	if err := s.sessionStore.RegisterConnection(clientIP, clientIP); err != nil {
		s.logger.Error("Failed to register connection", map[string]interface{}{
			"client": clientIP,
			"error":  err.Error(),
		})
	}

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
			currentID := s.pool.CurrentID(conn)

			s.logger.Info("TCP Packet Capture", map[string]interface{}{
				"client": currentID,
				"offset": fmt.Sprintf("0x%04X", totalBytes),
				"bytes":  n,
			})

			s.logger.Debug("Processing message", map[string]interface{}{
				"client": currentID,
				"bytes":  n,
			})

			response, err := s.messageHandler.HandleRawMessage(currentID, buffer[:n])
			if err != nil {
				s.logger.Error("Message handler error", map[string]interface{}{
					"client": currentID,
					"error":  err.Error(),
				})
			}

			if len(response) > 0 {
				// Re-fetch ID — may have been remapped during HandleRawMessage (e.g. Logon)
				writeID := s.pool.CurrentID(conn)
				if writeErr := s.pool.WriteToClient(writeID, response); writeErr != nil {
					s.logger.Error("Failed to send response", map[string]interface{}{
						"client": currentID,
						"error":  writeErr.Error(),
					})
					break
				}
			}

			totalBytes += n
		}
	}

	finalID := s.pool.CurrentID(conn)
	s.logger.Info("Connection closed", map[string]interface{}{
		"client":      finalID,
		"total_bytes": totalBytes,
	})
}

func (s *TCPServer) Shutdown() {
	s.logger.Info("Shutting down TCP server...")
	close(s.shutdown)

	if s.listener != nil {
		s.listener.Close()
	}

	s.pool.CloseAll()

	s.wg.Wait()
	s.logger.Info("TCP server shutdown complete")
}
