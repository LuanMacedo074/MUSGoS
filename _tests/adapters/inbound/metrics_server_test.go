package inbound_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
)

func TestMetricsServer_IncrementMethods(t *testing.T) {
	sessionStore := testutil.NewMockSessionStore()
	logger := &testutil.MockLogger{}

	ms := inbound.NewMetricsServer("0", "", sessionStore, logger)

	go ms.Start()
	defer ms.Shutdown()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Use the metrics server's counters
	ms.IncrementMessages()
	ms.IncrementMessages()
	ms.IncrementErrors()
	ms.IncrementRateLimited()
	ms.IncrementBannedConns()
}

func TestMetricsServer_Counters(t *testing.T) {
	sessionStore := testutil.NewMockSessionStore()
	logger := &testutil.MockLogger{}

	ms := inbound.NewMetricsServer("18932", "127.0.0.1", sessionStore, logger)

	go ms.Start()
	defer ms.Shutdown()

	time.Sleep(50 * time.Millisecond)

	ms.IncrementMessages()
	ms.IncrementMessages()
	ms.IncrementErrors()
	ms.IncrementRateLimited()
	ms.IncrementRateLimited()
	ms.IncrementRateLimited()
	ms.IncrementBannedConns()

	// Test /health
	resp, err := http.Get("http://localhost:18932/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", resp.StatusCode)
	}

	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)
	if health["status"] != "ok" {
		t.Errorf("health status = %v, want ok", health["status"])
	}

	// Test /metrics
	resp2, err := http.Get("http://localhost:18932/metrics")
	if err != nil {
		t.Fatalf("metrics request failed: %v", err)
	}
	defer resp2.Body.Close()

	var metrics map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&metrics)

	if metrics["messages_processed"] != float64(2) {
		t.Errorf("messages_processed = %v, want 2", metrics["messages_processed"])
	}
	if metrics["message_errors"] != float64(1) {
		t.Errorf("message_errors = %v, want 1", metrics["message_errors"])
	}
	if metrics["rate_limited"] != float64(3) {
		t.Errorf("rate_limited = %v, want 3", metrics["rate_limited"])
	}
	if metrics["banned_connections"] != float64(1) {
		t.Errorf("banned_connections = %v, want 1", metrics["banned_connections"])
	}
	if metrics["active_connections"] != float64(0) {
		t.Errorf("active_connections = %v, want 0", metrics["active_connections"])
	}
}
