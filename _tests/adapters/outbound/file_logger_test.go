package outbound_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func TestFileLogger_LevelFiltering(t *testing.T) {
	logDir := t.TempDir()
	logger := outbound.NewFileLogger("test", ports.WARN, logDir, 64)

	// These should be filtered out (below WARN)
	logger.Debug("debug msg")
	logger.Info("info msg")

	// These should be logged
	logger.Warn("warn msg")
	logger.Error("error msg")

	logger.Flush()

	// Read the log file
	content := readLogFile(t, logDir)

	if strings.Contains(content, "debug msg") {
		t.Error("DEBUG message should be filtered when level is WARN")
	}
	if strings.Contains(content, "info msg") {
		t.Error("INFO message should be filtered when level is WARN")
	}
	if !strings.Contains(content, "warn msg") {
		t.Error("WARN message should be present when level is WARN")
	}
	if !strings.Contains(content, "error msg") {
		t.Error("ERROR message should be present when level is WARN")
	}
}

func TestFileLogger_OutputFormat(t *testing.T) {
	logDir := t.TempDir()
	logger := outbound.NewFileLogger("mycomp", ports.DEBUG, logDir, 64)

	logger.Info("hello world", map[string]interface{}{"key": "val"})

	logger.Flush()

	content := readLogFile(t, logDir)

	if !strings.Contains(content, "mycomp") {
		t.Error("log should contain component name")
	}
	if !strings.Contains(content, "INFO") {
		t.Error("log should contain level")
	}
	if !strings.Contains(content, "hello world") {
		t.Error("log should contain message")
	}
	if !strings.Contains(content, "key=val") {
		t.Error("log should contain fields")
	}
}

func readLogFile(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no log files found")
	}

	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	return string(data)
}
