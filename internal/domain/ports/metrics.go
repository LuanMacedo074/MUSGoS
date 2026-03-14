package ports

type Metrics interface {
	IncrementMessages()
	IncrementErrors()
	IncrementRateLimited()
	IncrementBannedConns()
}
