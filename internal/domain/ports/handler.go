package ports

type MessageHandler interface {
	HandleRawMessage(clientID string, data []byte) ([]byte, error)
}
