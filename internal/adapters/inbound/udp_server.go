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

type UDPServer struct {
	config         UDPServerConfig
	conn           *net.UDPConn
	logger         ports.Logger
	shutdown       chan bool
	messageHandler ports.MessageHandler
}

func NewUDPServer(cfg UDPServerConfig, handler ports.MessageHandler, logger ports.Logger) *UDPServer {
	return &UDPServer{
		config:         cfg,
		messageHandler: handler,
		logger:         logger,
		shutdown:       make(chan bool),
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
