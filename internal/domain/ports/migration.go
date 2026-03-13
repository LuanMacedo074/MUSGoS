package ports

type Migration interface {
	Name() string
	Up(db DBAdapter) error
	Down(db DBAdapter) error
}

type MigrationTracker interface {
	GetAppliedMigrations() ([]string, error)
	MarkMigrationApplied(name string) error
}
