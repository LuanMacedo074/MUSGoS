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
	dispatcher   *mus.Dispatcher
	allEncrypted bool
}

func NewSMUSHandler(logger ports.Logger, cipher ports.Cipher, dispatcher *mus.Dispatcher, allEncrypted bool) *SMUSHandler {
	return &SMUSHandler{
		logger:       logger,
		cipher:       cipher,
		dispatcher:   dispatcher,
		allEncrypted: allEncrypted,
	}
}

func (h *SMUSHandler) HandleRawMessage(clientID string, data []byte) ([]byte, error) {
	h.logger.Debug("Processing SMUS message", map[string]interface{}{
		"client": clientID,
		"bytes":  len(data),
	})

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

	// For UDP connections, clientID is the remote address (not a session ID).
	// Use the message's SenderID if populated — the user logged in via TCP
	// and embedded their identity in the message.
	// Keep clientID for Logon messages since the Logon handler needs the
	// connection ID for session remap.
	dispatchID := clientID
	isLogon := msg.RecptID.Count > 0 && msg.RecptID.Strings[0].Value == "System" && msg.Subject.Value == "Logon"
	if !isLogon && msg.SenderID.Value != "" {
		dispatchID = msg.SenderID.Value
	}

	response, err := h.dispatcher.Dispatch(dispatchID, msg)
	if err != nil {
		return nil, err
	}
	if response != nil {
		responseBytes := response.GetBytes()
		if h.allEncrypted && h.cipher != nil {
			responseBytes = h.cipher.Encrypt(responseBytes)
		}
		return responseBytes, nil
	}
	return nil, nil
}
