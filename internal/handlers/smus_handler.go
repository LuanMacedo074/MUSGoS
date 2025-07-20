package handlers

import (
    "fsos-server/internal/types/smus"
    "fsos-server/internal/utilities/logger"
)

type SMUSHandler struct {
    logger *logger.Logger
}

func NewSMUSHandler(logger *logger.Logger) *SMUSHandler {
    return &SMUSHandler{
        logger: logger,
    }
}

func (h *SMUSHandler) HandleRawMessage(clientID string, data []byte) ([]byte, error) {
    h.logger.Debug("Processing SMUS message", map[string]interface{}{
        "client": clientID,
        "bytes":  len(data),
    })
    
    msg, err := smus.ParseMUSMessage(data)
    if err != nil {
        h.logger.Error("Failed to parse SMUS message", map[string]interface{}{
            "client": clientID,
            "error":  err.Error(),
            "bytes":  len(data),
        })
        return nil, err
    }
    
    h.logger.Info("SMUS Message Parsed", map[string]interface{}{
        "client":       clientID,
        "subject":      msg.Subject.Value,
        "sender":       msg.SenderID.Value,
        "recipients":   msg.RecptID.Count,
        "err_code":     msg.ErrCode,
        "timestamp":    msg.TimeStamp,
        "content_size": len(msg.RawContents),
    })
    
    h.logger.Debug("Message details", map[string]interface{}{
        "parsed": msg.String(),
    })
    
    return nil, nil
}