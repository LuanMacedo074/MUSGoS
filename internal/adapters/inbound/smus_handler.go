package inbound

import (
	"fmt"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/smus"
)

type SMUSHandler struct {
	logger       ports.Logger
	cipher       ports.Cipher
	scriptEngine ports.ScriptEngine
	logonService *mus.LogonService
}

func NewSMUSHandler(logger ports.Logger, cipher ports.Cipher, scriptEngine ports.ScriptEngine, logonService *mus.LogonService) *SMUSHandler {
	return &SMUSHandler{
		logger:       logger,
		cipher:       cipher,
		scriptEngine: scriptEngine,
		logonService: logonService,
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

	// Handle Logon messages
	if msg.Subject.Value == "Logon" && h.logonService != nil {
		response, err := h.logonService.HandleLogon(clientID, msg)
		if err != nil {
			h.logger.Error("Logon handling failed", map[string]interface{}{
				"client": clientID,
				"error":  err.Error(),
			})
			return nil, err
		}
		return response.GetBytes(), nil
	}

	// Route to script engine when recipient is "system.script"
	if h.scriptEngine != nil && h.hasRecipient(msg, "system.script") {
		scriptName := msg.Subject.Value
		if !h.scriptEngine.HasScript(scriptName) {
			h.logger.Warn("Script not found", map[string]interface{}{
				"client": clientID,
				"script": scriptName,
			})
			return nil, nil
		}

		scriptMsg := &ports.ScriptMessage{
			Subject:  scriptName,
			SenderID: msg.SenderID.Value,
			Content:  msg.MsgContent,
		}

		result, err := h.scriptEngine.Execute(scriptMsg)
		if err != nil {
			h.logger.Error("Script execution failed", map[string]interface{}{
				"client": clientID,
				"script": scriptName,
				"error":  err.Error(),
			})
			return nil, nil
		}

		h.logger.Debug("Script executed", map[string]interface{}{
			"client": clientID,
			"script": scriptName,
			"result": fmt.Sprintf("%v", result.Content),
		})

		if result.Content != nil {
			resp := mus.NewResponse(scriptName, "system.script", []string{msg.SenderID.Value}, smus.ErrNoError, result.Content)
			return resp.GetBytes(), nil
		}
	}

	return nil, nil
}

func (h *SMUSHandler) hasRecipient(msg *smus.MUSMessage, target string) bool {
	for _, r := range msg.RecptID.Strings {
		if r.Value == target {
			return true
		}
	}
	return false
}
