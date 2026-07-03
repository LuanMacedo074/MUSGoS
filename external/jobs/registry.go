// Package jobs holds recurring scheduled-job definitions. Each generated file
// registers a JobDefinition in its init(); the scheduler (wired in
// cmd/gameserver) runs each job's Lua script (external/scripts/jobs/<Name>.lua)
// on its interval. This is the time-driven counterpart to external/queues
// (which is message-driven) — the two systems are intentionally separate.
package jobs

// All is the registry of every job declared via Register (in init()).
var All []JobDefinition

// JobDefinition declares one recurring job. Name is the script basename under
// external/scripts/jobs/ (subject "jobs/<Name>"). IntervalSeconds is how often
// it runs. Enabled=false keeps the definition but skips scheduling it.
type JobDefinition struct {
	Name            string
	IntervalSeconds int
	Enabled         bool
}

func Register(j JobDefinition) {
	All = append(All, j)
}
