package inbound

import (
	"fmt"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/smus"
)

type SMUSHandler struct {
	logger       ports.Logger
	cipher       ports.Cipher
	scriptEngine ports.ScriptEngine
}

func NewSMUSHandler(logger ports.Logger, cipher ports.Cipher, scriptEngine ports.ScriptEngine) *SMUSHandler {
	return &SMUSHandler{
		logger:       logger,
		cipher:       cipher,
		scriptEngine: scriptEngine,
	}
}

func (h *SMUSHandler) HandleRawMessage(clientID string, data []byte) ([]byte, error) {
	h.logger.Debug("Processing SMUS message", map[string]interface{}{
		"client": clientID,
		"bytes":  len(data),
	})

	// Parse mensagem com descriptografia automática
	msg, err := smus.ParseMUSMessageWithDecryption(data, h.cipher)
	if err != nil {
		h.logger.Error("Failed to parse SMUS message", map[string]interface{}{
			"client": clientID,
			"error":  err.Error(),
			"bytes":  len(data),
		})
		return nil, err
	}

	h.logger.Debug("SMUS Message Parsed", map[string]interface{}{
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

	// Execute script if one exists for this subject
	if h.scriptEngine != nil && h.scriptEngine.HasScript(msg.Subject.Value) {
		scriptMsg := &ports.ScriptMessage{
			Subject:  msg.Subject.Value,
			SenderID: msg.SenderID.Value,
			Content:  msg.MsgContent,
		}

		// Script errors are intentionally non-fatal: a broken script should not
		// prevent the server from operating. The error is logged for debugging.
		result, err := h.scriptEngine.Execute(scriptMsg)
		if err != nil {
			h.logger.Error("Script execution failed", map[string]interface{}{
				"client":  clientID,
				"subject": msg.Subject.Value,
				"error":   err.Error(),
			})
		} else {
			h.logger.Debug("Script executed", map[string]interface{}{
				"client":  clientID,
				"subject": msg.Subject.Value,
				"result":  fmt.Sprintf("%v", result.Content),
			})
		}
	}

	return nil, nil
}
