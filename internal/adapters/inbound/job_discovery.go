package inbound

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"fsos-server/internal/domain/ports"
)

// jobHeaderRe matches a self-describing job header line in a job's Lua file:
//
//	-- @job interval=300
//	-- @job interval=2 enabled=false
//
// This keeps the job catalog fully data-driven: a new scheduled job is just a
// Lua file under external/scripts/jobs/ with this header — no Go registration.
var (
	jobHeaderRe   = regexp.MustCompile(`@job\b`)
	jobIntervalRe = regexp.MustCompile(`interval\s*=\s*(\d+)`)
	jobEnabledRe  = regexp.MustCompile(`enabled\s*=\s*(true|false)`)
)

// DiscoverJobs scans <scriptsDir>/jobs/*.lua, reads each file's "-- @job …"
// header, and returns the runnable ScheduledJob list. A job's Name is the file
// basename (subject "jobs/<name>"); its interval comes from interval=<seconds>;
// enabled defaults to true and enabled=false skips it. Files with no @job
// header or a non-positive interval are skipped (logged). Results are sorted by
// name for a deterministic start order.
func DiscoverJobs(scriptsDir string, logger ports.Logger) []ScheduledJob {
	dir := filepath.Join(scriptsDir, "jobs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if logger != nil {
			logger.Warn("DiscoverJobs: jobs directory not readable", map[string]interface{}{
				"dir":   dir,
				"error": err.Error(),
			})
		}
		return nil
	}

	var jobs []ScheduledJob
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".lua" {
			continue
		}
		name := e.Name()[:len(e.Name())-len(".lua")]
		interval, enabled, ok := parseJobHeader(filepath.Join(dir, e.Name()))
		if !ok {
			// No @job header — a helper/include, not a scheduled job.
			continue
		}
		if !enabled {
			continue
		}
		if interval <= 0 {
			if logger != nil {
				logger.Warn("DiscoverJobs: skipping job with non-positive interval", map[string]interface{}{
					"job": name,
				})
			}
			continue
		}
		jobs = append(jobs, ScheduledJob{Name: name, Interval: time.Duration(interval) * time.Second})
	}

	sort.Slice(jobs, func(i, j int) bool { return jobs[i].Name < jobs[j].Name })
	if logger != nil {
		names := make([]string, len(jobs))
		for i, j := range jobs {
			names[i] = j.Name
		}
		logger.Info("DiscoverJobs: scheduled jobs", map[string]interface{}{
			"count": len(jobs),
			"jobs":  names,
		})
	}
	return jobs
}

// parseJobHeader reads a job file's leading comment lines for the @job header.
// Returns (intervalSeconds, enabled, found). Only the first ~20 lines are read
// (the header is at the top).
func parseJobHeader(path string) (int, bool, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for i := 0; i < 20 && scanner.Scan(); i++ {
		line := scanner.Text()
		if !jobHeaderRe.MatchString(line) {
			continue
		}
		interval := 0
		if m := jobIntervalRe.FindStringSubmatch(line); m != nil {
			interval, _ = strconv.Atoi(m[1])
		}
		enabled := true
		if m := jobEnabledRe.FindStringSubmatch(line); m != nil {
			enabled = m[1] == "true"
		}
		return interval, enabled, true
	}
	return 0, false, false
}
