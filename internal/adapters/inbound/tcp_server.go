package inbound

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"

	"fsos-server/internal/domain/ports"
)

type TCPServer struct {
	port           string
	listener       net.Listener
	logger         ports.Logger
	shutdown       chan bool
	wg             sync.WaitGroup
	messageHandler ports.MessageHandler
	sessionStore   ports.SessionStore
}

func NewTCPServer(port string, logger ports.Logger, handler ports.MessageHandler, sessionStore ports.SessionStore) *TCPServer {
	return &TCPServer{
		port:           port,
		logger:         logger,
		shutdown:       make(chan bool),
		messageHandler: handler,
		sessionStore:   sessionStore,
	}
}

func (s *TCPServer) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.port, err)
	}

	s.logger.Info("TCP Server listening", map[string]interface{}{
		"port": s.port,
	})

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

	s.logger.Info("New connection established", map[string]interface{}{
		"client": clientIP,
	})

	reader := bufio.NewReader(conn)
	buffer := make([]byte, 4096)
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

	s.wg.Wait()
	s.logger.Info("TCP server shutdown complete")
}
