package handlers

import (
	"fmt"
	"fsos-server/internal/crypto"
	"fsos-server/internal/types/smus"
	"fsos-server/internal/utilities/logger"
)

type SMUSHandler struct {
	logger  *logger.Logger
	encrypt *crypto.Blowfish
}

func NewSMUSHandler(logger *logger.Logger, encrypt *crypto.Blowfish) *SMUSHandler {
	return &SMUSHandler{
		logger:  logger,
		encrypt: encrypt,
	}
}

func (h *SMUSHandler) HandleRawMessage(clientID string, data []byte) ([]byte, error) {
	h.logger.Debug("Processing SMUS message", map[string]interface{}{
		"client": clientID,
		"bytes":  len(data),
	})

	// Parse mensagem com descriptografia automática
	msg, err := smus.ParseMUSMessageWithDecryption(data, h.encrypt)
	if err != nil {
		h.logger.Error("Failed to parse SMUS message", map[string]interface{}{
			"client": clientID,
			"error":  err.Error(),
			"bytes":  len(data),
		})
		return nil, err
	}

	h.logger.Info("SMUS Message Parsed", map[string]interface{}{
		"client":           clientID,
		"subject":          msg.Subject.Value,
		"sender":           msg.SenderID.Value,
		"recipients":       msg.RecptID.Count,
		"err_code":         msg.ErrCode,
		"timestamp":        msg.TimeStamp,
		"raw_content_size": len(msg.RawContents),
		"decrypted_size":   len(msg.DecryptedContents),
	})

	if len(msg.DecryptedContents) > 0 {
		h.logger.Debug("Decrypted content details", map[string]interface{}{
			"client":           clientID,
			"decrypted_hex":    fmt.Sprintf("%X", msg.DecryptedContents),
			"decrypted_string": string(msg.DecryptedContents),
			"decrypted_length": len(msg.DecryptedContents),
		})
	}

	h.logger.Debug("Message details", map[string]interface{}{
		"parsed": msg.String(),
	})

	return nil, nil
}
