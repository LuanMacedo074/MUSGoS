package inbound

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/smus"
)

// musFrameHeaderLen is the fixed MUS envelope prefix: 2 magic bytes + 4-byte
// big-endian ContentSize. The full frame is musFrameHeaderLen + ContentSize.
const musFrameHeaderLen = 6

// nextFrame extracts one complete MUS message from the front of buf. A MUS frame
// is [0x72 0x00][ContentSize:uint32 BE][payload:ContentSize]. It returns ok=false
// with a nil error when more bytes are needed, and a non-nil error on a protocol
// violation (bad header / oversize frame) that must drop the connection.
func nextFrame(buf []byte, maxSize int) (frame, rest []byte, ok bool, err error) {
	if len(buf) < musFrameHeaderLen {
		return nil, buf, false, nil
	}
	if buf[0] != smus.MUSHeader[0] || buf[1] != smus.MUSHeader[1] {
		return nil, buf, false, fmt.Errorf("invalid MUS header: % X", buf[:2])
	}
	contentSize := int(binary.BigEndian.Uint32(buf[2:6]))
	if contentSize < 0 {
		return nil, buf, false, fmt.Errorf("negative content size: %d", contentSize)
	}
	total := musFrameHeaderLen + contentSize
	if maxSize > 0 && total > maxSize {
		return nil, buf, false, fmt.Errorf("frame size %d exceeds max %d", total, maxSize)
	}
	if len(buf) < total {
		return nil, buf, false, nil
	}
	return buf[:total], buf[total:], true, nil
}

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
	// OnDisconnect, if set, is called with the client's id when its connection
	// tears down (socket close, idle/kill-timer, admin delete, or shutdown).
	OnDisconnect func(clientID string)
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
	onDisconnect func(clientID string)
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
		onDisconnect: deps.OnDisconnect,
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

	// Absorb any panic from the parse/dispatch path so one malformed message
	// drops just this connection instead of crashing the whole process. Runs
	// last (LIFO), after the teardown defer below has already cleaned up.
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Recovered from panic in connection handler", map[string]interface{}{
				"client": clientIP,
				"panic":  fmt.Sprintf("%v", r),
			})
			if s.metrics != nil {
				s.metrics.IncrementErrors()
			}
		}
	}()

	s.pool.Register(conn, clientIP)

	defer func() {
		currentID := s.pool.Unregister(conn)

		// Flush hot-state before dropping the session (all teardown paths —
		// idle/kill-timer/admin-delete/shutdown — funnel through here).
		if s.onDisconnect != nil {
			s.onDisconnect(currentID)
		}

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
	readBuf := make([]byte, s.config.MaxMessageSize)
	// acc accumulates the byte stream; MUS is length-prefixed over TCP, so a
	// single Read may hold a partial message or several coalesced ones. We frame
	// on the [0x72 0x00][size] envelope rather than assuming one Read == one msg.
	var acc []byte
	totalBytes := 0

readLoop:
	for {
		n, err := reader.Read(readBuf)

		if n > 0 {
			totalBytes += n
			acc = append(acc, readBuf[:n]...)

			for {
				frame, rest, ok, frameErr := nextFrame(acc, s.config.MaxMessageSize)
				if frameErr != nil {
					s.logger.Error("Malformed frame; dropping connection", map[string]interface{}{
						"client": s.pool.CurrentID(conn),
						"error":  frameErr.Error(),
					})
					if s.metrics != nil {
						s.metrics.IncrementErrors()
					}
					break readLoop
				}
				if !ok {
					break // need more bytes for a complete frame
				}
				acc = rest

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

				s.logger.Debug("Processing message", map[string]interface{}{
					"client": currentID,
					"bytes":  len(frame),
				})

				response, herr := s.handler.HandleRawMessage(currentID, frame)
				if herr != nil {
					s.logger.Error("Message handler error", map[string]interface{}{
						"client": currentID,
						"error":  herr.Error(),
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
						break readLoop
					}
				}
			}
		}

		if err != nil {
			if err != io.EOF {
				s.logger.Error("Read error", map[string]interface{}{
					"client": clientIP,
					"error":  err,
				})
			}
			break
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
