package jobs

func init() {
	Register(JobDefinition{
		Name:            "heartbeat",
		IntervalSeconds: 300,
		Enabled:         true,
	})
}
