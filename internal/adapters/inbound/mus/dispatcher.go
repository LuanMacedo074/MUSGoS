package mus

import (
	"fmt"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/smus"
)

type Dispatcher struct {
	logger        ports.Logger
	scriptEngine  ports.ScriptEngine
	systemService *SystemService
	sender        *Sender
	queue         ports.QueuePublisher
}

func NewDispatcher(
	logger ports.Logger,
	scriptEngine ports.ScriptEngine,
	systemService *SystemService,
	sender *Sender,
	queue ports.QueuePublisher,
) *Dispatcher {
	return &Dispatcher{
		logger:        logger,
		scriptEngine:  scriptEngine,
		systemService: systemService,
		sender:        sender,
		queue:         queue,
	}
}

func (d *Dispatcher) Dispatch(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	if msg.RecptID.Count == 0 {
		return nil, fmt.Errorf("message has no recipients")
	}

	// MUS protocol routes by the first recipient only. The recipient list
	// is message metadata; routing decisions use the primary recipient.
	recipient := msg.RecptID.Strings[0].Value

	switch recipient {
	case "System":
		return d.systemService.Handle(senderID, msg)

	case "system.script":
		return d.handleScript(senderID, msg)

	default:
		// Group broadcast (@GroupName) or user-to-user
		if err := d.sender.SendMessage(senderID, recipient, msg.Subject.Value, msg.MsgContent); err != nil {
			d.logger.Error("Message delivery failed", map[string]interface{}{
				"senderID":  senderID,
				"recipient": recipient,
				"subject":   msg.Subject.Value,
				"error":     err.Error(),
			})
		}
		return nil, nil
	}
}

func (d *Dispatcher) handleScript(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	if d.scriptEngine == nil {
		return nil, nil
	}

	scriptName := msg.Subject.Value
	if !d.scriptEngine.HasScript(scriptName) {
		d.logger.Warn("Script not found", map[string]interface{}{
			"senderID": senderID,
			"script":   scriptName,
		})
		return nil, nil
	}

	scriptMsg := &ports.ScriptMessage{
		Subject:  scriptName,
		SenderID: senderID,
		Content:  msg.MsgContent,
	}

	result, err := d.scriptEngine.Execute(scriptMsg)
	if err != nil {
		d.logger.Error("Script execution failed", map[string]interface{}{
			"senderID": senderID,
			"script":   scriptName,
			"error":    err.Error(),
		})
		return nil, nil
	}

	d.logger.Debug("Script executed", map[string]interface{}{
		"senderID": senderID,
		"script":   scriptName,
		"result":   fmt.Sprintf("%v", result.Content),
	})

	if result.Content != nil {
		return NewResponse(scriptName, "system.script", []string{senderID}, smus.ErrNoError, result.Content), nil
	}

	return nil, nil
}
