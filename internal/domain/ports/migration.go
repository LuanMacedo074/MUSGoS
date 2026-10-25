package ports

// Migration is forward-only: there is no Down. A bad migration is fixed by
// adding a new migration that corrects it, never by rolling one back.
type Migration interface {
	Name() string
	Up(db DBAdapter) error
}

type MigrationTracker interface {
	GetAppliedMigrations() ([]string, error)
	MarkMigrationApplied(name string) error
}

// MigrationTx is a DBAdapter and MigrationTracker bound to one transaction:
// the runner applies a migration's Up and records it in the tracker through
// the same MigrationTx, so either both land or neither does.
type MigrationTx interface {
	DBAdapter
	MigrationTracker
	Commit() error
	Rollback() error
}

// MigrationTransactor is a DBAdapter that can open a MigrationTx. Adapters
// that can't simply omit it; the runner type-asserts for it and falls back to
// running Up and the tracker insert separately.
type MigrationTransactor interface {
	BeginMigration() (MigrationTx, error)
}
