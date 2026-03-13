package ports_test

import (
	"testing"

	"fsos-server/internal/domain/ports"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level ports.LogLevel
		want  string
	}{
		{ports.DEBUG, "DEBUG"},
		{ports.INFO, "INFO"},
		{ports.WARN, "WARN"},
		{ports.ERROR, "ERROR"},
		{ports.FATAL, "FATAL"},
		{ports.LogLevel(99), "UNKNOWN"},
		{ports.LogLevel(-1), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}
