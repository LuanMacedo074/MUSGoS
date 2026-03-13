package factory_test

import (
	"testing"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/factory"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  ports.LogLevel
	}{
		{"DEBUG", ports.DEBUG},
		{"INFO", ports.INFO},
		{"WARN", ports.WARN},
		{"ERROR", ports.ERROR},
		{"invalid", ports.INFO},
		{"", ports.INFO},
		{"debug", ports.INFO}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := factory.ParseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewLogger_File(t *testing.T) {
	logPath := t.TempDir()
	logger, err := factory.NewLogger("file", "test-component", ports.INFO, logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestNewLogger_Unknown(t *testing.T) {
	_, err := factory.NewLogger("console", "test", ports.INFO, "/tmp")
	if err == nil {
		t.Error("expected error for unknown logger type")
	}
}
