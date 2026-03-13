package ports

type ConnectionWriter interface {
	WriteToClient(clientID string, data []byte) error
	RemapClientID(oldID, newID string)
}
