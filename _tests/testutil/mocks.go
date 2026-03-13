package testutil

import "fsos-server/internal/domain/ports"

// LogEntry represents a single log call captured by MockLogger.
type LogEntry struct {
	Level  ports.LogLevel
	Msg    string
	Fields map[string]interface{}
}

// MockLogger implements ports.Logger and records all calls for inspection.
type MockLogger struct {
	Messages []LogEntry
}

func (m *MockLogger) Debug(msg string, fields ...map[string]interface{}) {
	m.record(ports.DEBUG, msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...map[string]interface{}) {
	m.record(ports.INFO, msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...map[string]interface{}) {
	m.record(ports.WARN, msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...map[string]interface{}) {
	m.record(ports.ERROR, msg, fields)
}

func (m *MockLogger) Fatal(msg string, fields ...map[string]interface{}) {
	m.record(ports.FATAL, msg, fields)
}

func (m *MockLogger) record(level ports.LogLevel, msg string, fields []map[string]interface{}) {
	entry := LogEntry{Level: level, Msg: msg}
	if len(fields) > 0 {
		entry.Fields = fields[0]
	}
	m.Messages = append(m.Messages, entry)
}

// MockCipher implements ports.Cipher with configurable behavior.
type MockCipher struct {
	DecryptFunc  func([]byte) []byte
	EncryptFunc  func([]byte) []byte
	DecryptCalls int
	EncryptCalls int
}

func (m *MockCipher) Encrypt(data []byte) []byte {
	m.EncryptCalls++
	if m.EncryptFunc != nil {
		return m.EncryptFunc(data)
	}
	return data
}

func (m *MockCipher) Decrypt(data []byte) []byte {
	m.DecryptCalls++
	if m.DecryptFunc != nil {
		return m.DecryptFunc(data)
	}
	return data
}
