package ports

type TimerManager interface {
	SetServerKillTimer(minutes int)
	CancelServerKillTimer()
	SetUserKillTimer(clientID string, minutes int)
	CancelUserKillTimer(clientID string)
	Stop()
}
