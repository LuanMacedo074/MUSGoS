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

type TCPServerDeps struct {
	Handler      ports.MessageHandler
	Pool         *ConnPool
	Logger       ports.Logger
	SessionStore ports.SessionStore
	BanChecker   *BanChecker
	RateLimiter  ports.RateLimiter
	Metrics      ports.Metrics
}

type TCPServer struct {
	config       TCPServerConfig
	listener     net.Listener
	shutdown     chan bool
	wg           sync.WaitGroup
	handler      ports.MessageHandler
	pool         *ConnPool
	logger       ports.Logger
	sessionStore ports.SessionStore
	banChecker   *BanChecker
	rateLimiter  ports.RateLimiter
	metrics      ports.Metrics
}

func NewTCPServer(cfg TCPServerConfig, deps TCPServerDeps) *TCPServer {
	return &TCPServer{
		config:       cfg,
		handler:      deps.Handler,
		pool:         deps.Pool,
		logger:       deps.Logger,
		shutdown:     make(chan bool),
		sessionStore: deps.SessionStore,
		banChecker:   deps.BanChecker,
		rateLimiter:  deps.RateLimiter,
		metrics:      deps.Metrics,
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

			host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
			if err != nil {
				s.logger.Error("Failed to parse remote address", map[string]interface{}{
					"error": err,
				})
				conn.Close()
				continue
			}

			if s.banChecker != nil && s.banChecker.IsIPBanned(host) {
				s.logger.Info("Connection rejected: IP is banned", map[string]interface{}{
					"ip": host,
				})
				if s.metrics != nil {
					s.metrics.IncrementBannedConns()
				}
				conn.Close()
				continue
			}

			s.wg.Add(1)
			go s.handleConnection(conn, host)
		}
	}
}

func (s *TCPServer) handleConnection(conn net.Conn, host string) {
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

			if s.rateLimiter != nil && !s.rateLimiter.Allow(host) {
				s.logger.Warn("Rate limit exceeded", map[string]interface{}{
					"client": currentID,
				})
				if s.metrics != nil {
					s.metrics.IncrementRateLimited()
				}
				continue
			}

			s.logger.Info("TCP Packet Capture", map[string]interface{}{
				"client": currentID,
				"offset": fmt.Sprintf("0x%04X", totalBytes),
				"bytes":  n,
			})

			s.logger.Debug("Processing message", map[string]interface{}{
				"client": currentID,
				"bytes":  n,
			})

			response, err := s.handler.HandleRawMessage(currentID, buffer[:n])
			if err != nil {
				s.logger.Error("Message handler error", map[string]interface{}{
					"client": currentID,
					"error":  err.Error(),
				})
				if s.metrics != nil {
					s.metrics.IncrementErrors()
				}
			} else {
				s.sessionStore.UpdateLastActivity(currentID)
				if s.metrics != nil {
					s.metrics.IncrementMessages()
				}
			}

			if len(response) > 0 {
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
