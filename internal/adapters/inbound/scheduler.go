package inbound

import (
	"sync"
	"time"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/types/lingo"
)

// jobSenderID is the synthetic sender used for scheduler-invoked scripts. Jobs
// operate server-wide and derive any per-player target from their own DB
// queries; getSender() in a job script returns this label.
const jobSenderID = "system.jobs"

// ScheduledJob is one recurring job: a Lua script under external/scripts/jobs/
// run every Interval. Name is the script basename (subject "jobs/<Name>").
type ScheduledJob struct {
	Name     string
	Interval time.Duration
}

// Scheduler runs registered jobs on their own tickers, invoking the shared
// script engine once per interval. It is a standalone background service
// (separate from the message-queue/consumer system) modeled on IdleChecker:
// one goroutine per job, a shared done channel, and a once-guarded Stop.
type Scheduler struct {
	engine   ports.ScriptEngine
	logger   ports.Logger
	jobs     []ScheduledJob
	done     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// NewScheduler builds a scheduler for the given jobs. Jobs with a non-positive
// interval or whose script is missing are dropped at construction (logged), so
// Start only ticks runnable jobs. A nil engine yields a no-op scheduler.
func NewScheduler(engine ports.ScriptEngine, jobs []ScheduledJob, logger ports.Logger) *Scheduler {
	var runnable []ScheduledJob
	for _, j := range jobs {
		if j.Interval <= 0 {
			logger.Warn("Scheduler: skipping job with non-positive interval", map[string]interface{}{
				"job":      j.Name,
				"interval": j.Interval.String(),
			})
			continue
		}
		if engine == nil || !engine.HasScript("jobs/"+j.Name) {
			logger.Warn("Scheduler: skipping job with no script", map[string]interface{}{
				"job":    j.Name,
				"script": "jobs/" + j.Name + ".lua",
			})
			continue
		}
		runnable = append(runnable, j)
	}

	return &Scheduler{
		engine: engine,
		logger: logger,
		jobs:   runnable,
		done:   make(chan struct{}),
	}
}

// Start launches one goroutine per job. Each ticks on its own interval and runs
// the job until Stop is called. No-op when there are no runnable jobs.
func (s *Scheduler) Start() {
	if len(s.jobs) == 0 {
		s.logger.Info("Scheduler: no jobs to run")
		return
	}

	names := make([]string, len(s.jobs))
	for i, j := range s.jobs {
		names[i] = j.Name
		s.wg.Add(1)
		go s.runJobLoop(j)
	}
	s.logger.Info("Scheduler started", map[string]interface{}{
		"count": len(s.jobs),
		"jobs":  names,
	})
}

func (s *Scheduler) runJobLoop(job ScheduledJob) {
	defer s.wg.Done()
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.runJob(job)
		}
	}
}

// runJob executes a single job's script, recovering from panics so one bad job
// can never take down the scheduler goroutine.
func (s *Scheduler) runJob(job ScheduledJob) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Scheduler: job panicked", map[string]interface{}{
				"job":   job.Name,
				"panic": r,
			})
		}
	}()

	_, err := s.engine.Execute(&ports.ScriptMessage{
		Subject:  "jobs/" + job.Name,
		SenderID: jobSenderID,
		Content:  lingo.NewLVoid(),
	})
	if err != nil {
		s.logger.Error("Scheduler: job failed", map[string]interface{}{
			"job":   job.Name,
			"error": err.Error(),
		})
	}
}

// Stop signals every job goroutine to exit and waits for them to finish.
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)
		s.wg.Wait()
		s.logger.Info("Scheduler stopped")
	})
}
