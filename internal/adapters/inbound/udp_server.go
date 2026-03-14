package inbound

import (
	"fmt"
	"net"

	"fsos-server/internal/domain/ports"
)

type UDPServerConfig struct {
	Port           string
	ServerIP       string
	MaxMessageSize int
}

type UDPServerDeps struct {
	Handler     ports.MessageHandler
	Logger      ports.Logger
	BanChecker  *BanChecker
	RateLimiter ports.RateLimiter
	Metrics     ports.Metrics
}

type UDPServer struct {
	config         UDPServerConfig
	conn           *net.UDPConn
	shutdown       chan bool
	logger         ports.Logger
	messageHandler ports.MessageHandler
	banChecker     *BanChecker
	rateLimiter    ports.RateLimiter
	metrics        ports.Metrics
}

func NewUDPServer(cfg UDPServerConfig, deps UDPServerDeps) *UDPServer {
	return &UDPServer{
		config:         cfg,
		messageHandler: deps.Handler,
		logger:         deps.Logger,
		shutdown:       make(chan bool),
		banChecker:     deps.BanChecker,
		rateLimiter:    deps.RateLimiter,
		metrics:        deps.Metrics,
	}
}

func (s *UDPServer) Start(ready chan struct{}) error {
	addr := s.config.ServerIP + ":" + s.config.Port
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", addr, err)
	}

	s.conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen UDP on %s: %w", addr, err)
	}

	s.logger.Info("UDP Server listening", map[string]interface{}{
		"address": addr,
	})

	if ready != nil {
		close(ready)
	}

	buf := make([]byte, s.config.MaxMessageSize)
	for {
		n, remoteAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-s.shutdown:
				return nil
			default:
				s.logger.Error("UDP read error", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}
		}

		if n > 0 {
			s.handlePacket(buf[:n], remoteAddr)
		}
	}
}

func (s *UDPServer) handlePacket(data []byte, addr *net.UDPAddr) {
	clientIP := addr.IP.String()

	if s.banChecker != nil && s.banChecker.IsIPBanned(clientIP) {
		if s.metrics != nil {
			s.metrics.IncrementBannedConns()
		}
		return
	}

	if s.rateLimiter != nil && !s.rateLimiter.Allow(clientIP) {
		s.logger.Warn("UDP rate limit exceeded", map[string]interface{}{
			"client": clientIP,
		})
		if s.metrics != nil {
			s.metrics.IncrementRateLimited()
		}
		return
	}

	clientID := addr.String()

	s.logger.Debug("UDP packet received", map[string]interface{}{
		"client": clientID,
		"bytes":  len(data),
	})

	response, err := s.messageHandler.HandleRawMessage(clientID, data)
	if err != nil {
		s.logger.Error("UDP message handler error", map[string]interface{}{
			"client": clientID,
			"error":  err.Error(),
		})
		if s.metrics != nil {
			s.metrics.IncrementErrors()
		}
	} else if s.metrics != nil {
		s.metrics.IncrementMessages()
	}

	if len(response) > 0 {
		if _, writeErr := s.conn.WriteToUDP(response, addr); writeErr != nil {
			s.logger.Error("UDP write error", map[string]interface{}{
				"client": clientID,
				"error":  writeErr.Error(),
			})
		}
	}
}

func (s *UDPServer) Shutdown() {
	s.logger.Info("Shutting down UDP server...")
	close(s.shutdown)
	if s.conn != nil {
		s.conn.Close()
	}
	s.logger.Info("UDP server shutdown complete")
}
