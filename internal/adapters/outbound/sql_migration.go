package outbound

import (
	"fmt"

	"fsos-server/internal/domain/ports"
)

// ExecuteMigrationSQL runs the current backend's trusted DDL inside the
// runner's migration transaction, so it commits or rolls back with Up and the
// tracker row. It opens no transaction of its own: each call is wrapped in a
// savepoint, so a failing statement undoes only this call's statements.
func (t *sqlMigrationTx) ExecuteMigrationSQL(ddl ports.MigrationSQL) error {
	var statements []string
	switch t.dialect.(type) {
	case sqliteDialect:
		statements = ddl.SQLite
	case postgresDialect:
		statements = ddl.Postgres
	default:
		return fmt.Errorf("SQL migrations are unsupported for %T", t.dialect)
	}
	sp, err := beginSavepoint(t.tx, t.dialect)
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := t.tx.Exec(statement); err != nil {
			if rbErr := sp.Rollback(); rbErr != nil {
				return fmt.Errorf("%w (rollback to savepoint failed: %v)", err, rbErr)
			}
			return err
		}
	}
	return sp.Commit()
}
