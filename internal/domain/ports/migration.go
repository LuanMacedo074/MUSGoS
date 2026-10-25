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

// MigrationSQL holds trusted, backend-specific DDL that the portable schema
// builders cannot express (for example, cross-table validation triggers).
// This optional migration capability is not exposed to scripts or query callers.
type MigrationSQL struct {
	SQLite   []string
	Postgres []string
}

// SQLMigrationExecutor runs the current backend's MigrationSQL statements. Only
// the MigrationTx the runner hands a migration implements it, so the
// statements always land in the runner's transaction with Up and the tracker
// row. Each call is scoped by a savepoint: a failing statement undoes the
// call's earlier statements and leaves the transaction usable.
type SQLMigrationExecutor interface {
	ExecuteMigrationSQL(MigrationSQL) error
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
