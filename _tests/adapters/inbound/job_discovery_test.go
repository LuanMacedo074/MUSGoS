package inbound_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
)

// DiscoverJobs scans <scriptsDir>/jobs/*.lua and reads each file's "-- @job"
// header (interval + optional enabled), so a scheduled job is pure Lua with no
// Go registration.
func TestDiscoverJobs_ReadsHeaders(t *testing.T) {
	scriptsDir := t.TempDir()
	jobsDir := filepath.Join(scriptsDir, "jobs")
	if err := os.MkdirAll(jobsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(jobsDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("heartbeat.lua", "-- @job interval=300\nmus.log.info('hi')\n")
	write("fast.lua", "-- @job interval=2 enabled=true\n-- do stuff\n")
	write("off.lua", "-- @job interval=5 enabled=false\n")
	write("helper.lua", "-- a shared helper, NOT a job (no @job header)\nreturn {}\n")
	write("bad.lua", "-- @job interval=0\n") // non-positive interval skipped

	jobs := inbound.DiscoverJobs(scriptsDir, &testutil.MockLogger{})

	got := map[string]time.Duration{}
	for _, j := range jobs {
		got[j.Name] = j.Interval
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 runnable jobs, got %d: %v", len(got), got)
	}
	if got["heartbeat"] != 300*time.Second {
		t.Errorf("heartbeat interval = %v, want 300s", got["heartbeat"])
	}
	if got["fast"] != 2*time.Second {
		t.Errorf("fast interval = %v, want 2s", got["fast"])
	}
	if _, ok := got["off"]; ok {
		t.Error("enabled=false job must be skipped")
	}
	if _, ok := got["helper"]; ok {
		t.Error("no-@job helper must be skipped")
	}
	if _, ok := got["bad"]; ok {
		t.Error("non-positive interval must be skipped")
	}
}

func TestDiscoverJobs_MissingDir(t *testing.T) {
	if jobs := inbound.DiscoverJobs(t.TempDir(), &testutil.MockLogger{}); jobs != nil {
		t.Fatalf("expected nil for missing jobs dir, got %v", jobs)
	}
}
