package inbound_test

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// countingEngine records every Execute call (subject + sender) and counts them.
type countingEngine struct {
	has      func(subject string) bool
	calls    int32
	subjects sync.Map
}

func (e *countingEngine) HasScript(subject string) bool {
	if e.has != nil {
		return e.has(subject)
	}
	return true
}

func (e *countingEngine) Execute(msg *ports.ScriptMessage) (*ports.ScriptResult, error) {
	atomic.AddInt32(&e.calls, 1)
	e.subjects.Store(msg.Subject, msg.SenderID)
	return &ports.ScriptResult{Content: lingo.NewLVoid()}, nil
}

func TestScheduler_RunsJobOnInterval(t *testing.T) {
	engine := &countingEngine{}
	log := &testutil.MockLogger{}

	s := inbound.NewScheduler(engine, []inbound.ScheduledJob{
		{Name: "tick", Interval: 20 * time.Millisecond},
	}, log)
	s.Start()

	// Let it fire a few times, then stop.
	time.Sleep(110 * time.Millisecond)
	s.Stop()

	got := atomic.LoadInt32(&engine.calls)
	if got < 3 {
		t.Fatalf("expected the job to fire at least 3 times in ~110ms, got %d", got)
	}

	// It ran the right subject with the synthetic sender.
	sender, ok := engine.subjects.Load("jobs/tick")
	if !ok {
		t.Fatal("expected subject jobs/tick to have been executed")
	}
	if sender != "system.jobs" {
		t.Fatalf("expected sender system.jobs, got %v", sender)
	}
}

func TestScheduler_StopHaltsExecution(t *testing.T) {
	engine := &countingEngine{}
	log := &testutil.MockLogger{}

	s := inbound.NewScheduler(engine, []inbound.ScheduledJob{
		{Name: "tick", Interval: 20 * time.Millisecond},
	}, log)
	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Stop()

	after := atomic.LoadInt32(&engine.calls)
	time.Sleep(80 * time.Millisecond)
	if atomic.LoadInt32(&engine.calls) != after {
		t.Fatalf("expected no further executions after Stop; before=%d after=%d", after, atomic.LoadInt32(&engine.calls))
	}
}

func TestScheduler_StopIsIdempotent(t *testing.T) {
	engine := &countingEngine{}
	log := &testutil.MockLogger{}

	s := inbound.NewScheduler(engine, []inbound.ScheduledJob{
		{Name: "tick", Interval: 20 * time.Millisecond},
	}, log)
	s.Start()
	s.Stop()
	s.Stop() // must not panic on double close
}

func TestScheduler_SkipsMissingScript(t *testing.T) {
	engine := &countingEngine{has: func(subject string) bool { return subject == "jobs/real" }}
	log := &testutil.MockLogger{}

	s := inbound.NewScheduler(engine, []inbound.ScheduledJob{
		{Name: "real", Interval: 20 * time.Millisecond},
		{Name: "ghost", Interval: 20 * time.Millisecond},
	}, log)
	s.Start()
	time.Sleep(60 * time.Millisecond)
	s.Stop()

	engine.subjects.Range(func(k, _ any) bool {
		if k == "jobs/ghost" {
			t.Fatal("missing-script job jobs/ghost should never execute")
		}
		return true
	})
}

func TestScheduler_SkipsNonPositiveInterval(t *testing.T) {
	engine := &countingEngine{}
	log := &testutil.MockLogger{}

	s := inbound.NewScheduler(engine, []inbound.ScheduledJob{
		{Name: "zero", Interval: 0},
	}, log)
	s.Start()
	time.Sleep(40 * time.Millisecond)
	s.Stop()

	if atomic.LoadInt32(&engine.calls) != 0 {
		t.Fatalf("expected no executions for non-positive interval, got %d", atomic.LoadInt32(&engine.calls))
	}
}

func TestScheduler_NilEngineIsNoOp(t *testing.T) {
	log := &testutil.MockLogger{}
	s := inbound.NewScheduler(nil, []inbound.ScheduledJob{
		{Name: "tick", Interval: 20 * time.Millisecond},
	}, log)
	s.Start() // must not panic
	s.Stop()
}

// End-to-end: the scheduler drives the REAL Lua engine, which loads and runs an
// actual jobs/<name>.lua that touches mus.log and mus.server — the full stack a
// migrated Fase-10 timer will use.
func TestScheduler_RunsRealLuaJobEndToEnd(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "jobs"), 0o755); err != nil {
		t.Fatalf("mkdir jobs: %v", err)
	}
	// Mirrors external/scripts/jobs/heartbeat.lua.
	script := `mus.log.info("scheduler heartbeat", "users_online", mus.server.getUserCount())`
	if err := os.WriteFile(filepath.Join(dir, "jobs", "heartbeat.lua"), []byte(script), 0o644); err != nil {
		t.Fatalf("write job: %v", err)
	}

	// Separate loggers: the engine's is written only by the job goroutine and
	// read after Stop() (which joins it), so there's no cross-goroutine sharing.
	engineLog := &testutil.MockLogger{}
	engine := outbound.NewLuaScriptEngine(dir, engineLog, 5, nil, nil, nil, nil, testutil.NewMockSessionStore(), nil, nil)

	s := inbound.NewScheduler(engine, []inbound.ScheduledJob{
		{Name: "heartbeat", Interval: 25 * time.Millisecond},
	}, &testutil.MockLogger{})
	s.Start()
	time.Sleep(80 * time.Millisecond)
	s.Stop()

	found := false
	for _, e := range engineLog.Messages {
		if e.Msg == "scheduler heartbeat" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected the real jobs/heartbeat.lua to have logged 'scheduler heartbeat'")
	}
}
