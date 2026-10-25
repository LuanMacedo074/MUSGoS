package outbound_test

import (
	"errors"
	"testing"

	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
)

// Both SQL adapters must apply a migration and its tracker row atomically. The
// scenarios below run once per backend (TestSQLite_Migration* / TestPG_Migration*).

const migTxTable = "mig_tx_test"

// migrationDB is what these scenarios need from an adapter under test.
type migrationDB interface {
	ports.DBAdapter
	ports.MigrationTracker
	QueryBuilder() ports.QueryBuilder
}

// funcMigration is a migration whose Up is a closure.
type funcMigration struct {
	name string
	up   func(db ports.DBAdapter) error
}

func (m *funcMigration) Name() string                { return m.name }
func (m *funcMigration) Up(db ports.DBAdapter) error { return m.up(db) }

func createMigTxTable(db ports.DBAdapter) error {
	return db.CreateTable(ports.Table{
		Name: migTxTable,
		Columns: []ports.Column{
			ports.Col("k", ports.ColText).NotNull(),
		},
		PrimaryKeys: []string{"k"},
	})
}

// beginOwnTx is how a migration opens its own transaction: through the query
// builder of the adapter it was handed.
func beginOwnTx(t *testing.T, db ports.DBAdapter) ports.Tx {
	t.Helper()
	qbp, ok := db.(interface{ QueryBuilder() ports.QueryBuilder })
	if !ok {
		t.Fatal("migration adapter must expose QueryBuilder()")
	}
	txqb, ok := qbp.QueryBuilder().(ports.TransactionalQueryBuilder)
	if !ok {
		t.Fatal("migration adapter's query builder must be transactional")
	}
	tx, err := txqb.Begin()
	if err != nil {
		t.Fatalf("begin own tx: %v", err)
	}
	return tx
}

func isApplied(t *testing.T, db migrationDB, name string) bool {
	t.Helper()
	applied, err := db.GetAppliedMigrations()
	mustNoErr(t, err)
	for _, a := range applied {
		if a == name {
			return true
		}
	}
	return false
}

func tableExists(db migrationDB, table string) bool {
	_, err := db.QueryBuilder().Table(table).Count()
	return err == nil
}

func runMigrations(db migrationDB, ms ...ports.Migration) error {
	_, err := services.NewMigrationRunner(db, db, ms).RunPending()
	return err
}

func checkFailingUpLeavesNothing(t *testing.T, db migrationDB) {
	const name = "29990101000000_failing"
	err := runMigrations(db, &funcMigration{name: name, up: func(tx ports.DBAdapter) error {
		if err := createMigTxTable(tx); err != nil {
			return err
		}
		return errors.New("boom after DDL")
	}})
	if err == nil {
		t.Fatal("expected the failing migration to surface its error")
	}
	if tableExists(db, migTxTable) {
		t.Error("failing Up must leave no partial DDL behind")
	}
	if isApplied(t, db, name) {
		t.Error("failing Up must leave no tracker row")
	}

	// The migration can be fixed and re-run without colliding with leftovers.
	mustNoErr(t, runMigrations(db, &funcMigration{name: name, up: createMigTxTable}))
	if !tableExists(db, migTxTable) || !isApplied(t, db, name) {
		t.Error("re-run of the fixed migration must create the table and record it")
	}
}

func checkOwnTransactionCommits(t *testing.T, db migrationDB) {
	const name = "29990102000000_own_tx"
	mustNoErr(t, runMigrations(db, &funcMigration{name: name, up: func(mdb ports.DBAdapter) error {
		if err := createMigTxTable(mdb); err != nil {
			return err
		}
		tx := beginOwnTx(t, mdb)
		if err := tx.Table(migTxTable).Insert(map[string]interface{}{"k": "kept"}); err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit()
	}}))

	if n := countRows(t, db, migTxTable); n != 1 {
		t.Errorf("expected the migration's own committed tx to persist 1 row, got %d", n)
	}
	if !isApplied(t, db, name) {
		t.Error("migration with its own transaction must be recorded")
	}
}

func checkOwnTransactionRollbackIsScoped(t *testing.T, db migrationDB) {
	const name = "29990103000000_own_tx_rollback"
	mustNoErr(t, runMigrations(db, &funcMigration{name: name, up: func(mdb ports.DBAdapter) error {
		if err := createMigTxTable(mdb); err != nil {
			return err
		}
		tx := beginOwnTx(t, mdb)
		if err := tx.Table(migTxTable).Insert(map[string]interface{}{"k": "discarded"}); err != nil {
			tx.Rollback()
			return err
		}
		return tx.Rollback()
	}}))

	if !tableExists(db, migTxTable) {
		t.Error("rolling back the migration's own tx must not undo the rest of the migration")
	}
	if n := countRows(t, db, migTxTable); n != 0 {
		t.Errorf("rolled-back rows must not persist, got %d", n)
	}
	if !isApplied(t, db, name) {
		t.Error("migration must still be recorded")
	}
}

func checkOwnTransactionUndoneWhenUpFails(t *testing.T, db migrationDB) {
	const name = "29990104000000_own_tx_then_fail"
	err := runMigrations(db, &funcMigration{name: name, up: func(mdb ports.DBAdapter) error {
		if err := createMigTxTable(mdb); err != nil {
			return err
		}
		tx := beginOwnTx(t, mdb)
		if err := tx.Table(migTxTable).Insert(map[string]interface{}{"k": "x"}); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return errors.New("boom after inner commit")
	}})
	if err == nil {
		t.Fatal("expected error")
	}
	if tableExists(db, migTxTable) || isApplied(t, db, name) {
		t.Error("an inner commit must not escape the migration's transaction when Up fails")
	}
}

// sqlExecutor returns the raw-SQL capability of the adapter a migration was
// handed; the runner's migration transaction must provide it.
func sqlExecutor(db ports.DBAdapter) (ports.SQLMigrationExecutor, error) {
	executor, ok := db.(ports.SQLMigrationExecutor)
	if !ok {
		return nil, errors.New("migration adapter cannot execute SQL migrations")
	}
	return executor, nil
}

// migTxSQL creates migTxTable on either backend.
var migTxSQL = ports.MigrationSQL{
	SQLite:   []string{"CREATE TABLE " + migTxTable + " (k TEXT PRIMARY KEY)"},
	Postgres: []string{"CREATE TABLE " + migTxTable + " (k TEXT PRIMARY KEY)"},
}

func checkRootAdapterHasNoSQLExecutor(t *testing.T, db migrationDB) {
	if _, ok := db.(ports.SQLMigrationExecutor); ok {
		t.Error("raw migration SQL must only run inside the runner's migration transaction")
	}
}

func checkSQLStatementsCommitWithMigration(t *testing.T, db migrationDB) {
	const name = "29990105000000_sql"
	mustNoErr(t, runMigrations(db, &funcMigration{name: name, up: func(mdb ports.DBAdapter) error {
		executor, err := sqlExecutor(mdb)
		if err != nil {
			return err
		}
		return executor.ExecuteMigrationSQL(ports.MigrationSQL{
			SQLite:   append(migTxSQL.SQLite, "INSERT INTO "+migTxTable+" (k) VALUES ('sqlite')"),
			Postgres: append(migTxSQL.Postgres, "INSERT INTO "+migTxTable+" (k) VALUES ('postgres')"),
		})
	}}))
	if !isApplied(t, db, name) {
		t.Error("migration running raw SQL must be recorded")
	}
	if n := countRows(t, db, migTxTable); n != 1 {
		t.Fatalf("expected the dialect's statements to run once, got %d rows", n)
	}
}

func checkSQLStatementsRunForTheBackendOnly(t *testing.T, db migrationDB, want string) {
	checkSQLStatementsCommitWithMigration(t, db)
	row, err := db.QueryBuilder().Table(migTxTable).First()
	mustNoErr(t, err)
	if row == nil || row["k"] != want {
		t.Errorf("expected only the %s statements to run, got %v", want, row)
	}
}

func checkSQLStatementsRollBackWithFailedUp(t *testing.T, db migrationDB) {
	const name = "29990106000000_sql_then_fail"
	err := runMigrations(db, &funcMigration{name: name, up: func(mdb ports.DBAdapter) error {
		executor, err := sqlExecutor(mdb)
		if err != nil {
			return err
		}
		if err := executor.ExecuteMigrationSQL(migTxSQL); err != nil {
			return err
		}
		return errors.New("failure after SQL DDL")
	}})
	if err == nil {
		t.Fatal("expected upgrade failure")
	}
	if tableExists(db, migTxTable) || isApplied(t, db, name) {
		t.Error("SQL DDL and tracker must roll back with the migration")
	}
}

// A failing statement undoes only its own call: the migration may handle the
// error and go on, and the runner's transaction stays usable (on Postgres an
// error would otherwise abort it).
func checkFailedSQLCallIsScoped(t *testing.T, db migrationDB) {
	const name = "29990107000000_sql_call_fails"
	mustNoErr(t, runMigrations(db, &funcMigration{name: name, up: func(mdb ports.DBAdapter) error {
		executor, err := sqlExecutor(mdb)
		if err != nil {
			return err
		}
		failing := ports.MigrationSQL{
			SQLite:   []string{"CREATE TABLE mig_tx_partial (k TEXT)", "SELECT * FROM mig_tx_missing"},
			Postgres: []string{"CREATE TABLE mig_tx_partial (k TEXT)", "SELECT * FROM mig_tx_missing"},
		}
		if err := executor.ExecuteMigrationSQL(failing); err == nil {
			return errors.New("expected the failing statement to surface its error")
		}
		return executor.ExecuteMigrationSQL(migTxSQL)
	}}))
	if tableExists(db, "mig_tx_partial") {
		t.Error("a failed SQL call must undo its earlier statements")
	}
	if !tableExists(db, migTxTable) || !isApplied(t, db, name) {
		t.Error("the migration must go on after handling a failed SQL call")
	}
}

func countRows(t *testing.T, db migrationDB, table string) int64 {
	t.Helper()
	n, err := db.QueryBuilder().Table(table).Count()
	mustNoErr(t, err)
	return n
}

func TestSQLite_Migration_FailingUpLeavesNoDDLNorTrackerRow(t *testing.T) {
	checkFailingUpLeavesNothing(t, newTestDB(t))
}

func TestSQLite_Migration_OwnTransactionCommits(t *testing.T) {
	checkOwnTransactionCommits(t, newTestDB(t))
}

func TestSQLite_Migration_OwnTransactionRollbackIsScoped(t *testing.T) {
	checkOwnTransactionRollbackIsScoped(t, newTestDB(t))
}

func TestSQLite_Migration_OwnTransactionUndoneWhenUpFails(t *testing.T) {
	checkOwnTransactionUndoneWhenUpFails(t, newTestDB(t))
}

func TestPG_Migration_FailingUpLeavesNoDDLNorTrackerRow(t *testing.T) {
	checkFailingUpLeavesNothing(t, newTestPG(t))
}

func TestPG_Migration_OwnTransactionCommits(t *testing.T) {
	checkOwnTransactionCommits(t, newTestPG(t))
}

func TestPG_Migration_OwnTransactionRollbackIsScoped(t *testing.T) {
	checkOwnTransactionRollbackIsScoped(t, newTestPG(t))
}

func TestPG_Migration_OwnTransactionUndoneWhenUpFails(t *testing.T) {
	checkOwnTransactionUndoneWhenUpFails(t, newTestPG(t))
}

func TestSQLite_Migration_RootAdapterHasNoSQLExecutor(t *testing.T) {
	checkRootAdapterHasNoSQLExecutor(t, newTestDB(t))
}

func TestSQLite_Migration_SQLStatementsRunForTheBackendOnly(t *testing.T) {
	checkSQLStatementsRunForTheBackendOnly(t, newTestDB(t), "sqlite")
}

func TestSQLite_Migration_SQLStatementsRollBackWithFailedUp(t *testing.T) {
	checkSQLStatementsRollBackWithFailedUp(t, newTestDB(t))
}

func TestSQLite_Migration_FailedSQLCallIsScoped(t *testing.T) {
	checkFailedSQLCallIsScoped(t, newTestDB(t))
}

func TestPG_Migration_RootAdapterHasNoSQLExecutor(t *testing.T) {
	checkRootAdapterHasNoSQLExecutor(t, newTestPG(t))
}

func TestPG_Migration_SQLStatementsRunForTheBackendOnly(t *testing.T) {
	checkSQLStatementsRunForTheBackendOnly(t, newTestPG(t), "postgres")
}

func TestPG_Migration_SQLStatementsRollBackWithFailedUp(t *testing.T) {
	checkSQLStatementsRollBackWithFailedUp(t, newTestPG(t))
}

func TestPG_Migration_FailedSQLCallIsScoped(t *testing.T) {
	checkFailedSQLCallIsScoped(t, newTestPG(t))
}

func TestPG_Migration_QueriesSurviveTableShapeChange(t *testing.T) {
	db := newTestPG(t)
	read := func(mdb ports.DBAdapter) error {
		qb := mdb.(interface{ QueryBuilder() ports.QueryBuilder }).QueryBuilder()
		_, err := qb.Table(migTxTable).First()
		return err
	}
	mustNoErr(t, runMigrations(db,
		&funcMigration{name: "29990201000000_create_and_read", up: func(mdb ports.DBAdapter) error {
			if err := createMigTxTable(mdb); err != nil {
				return err
			}
			qb := mdb.(interface{ QueryBuilder() ports.QueryBuilder }).QueryBuilder()
			if err := qb.Table(migTxTable).Insert(map[string]interface{}{"k": "kept"}); err != nil {
				return err
			}
			return read(mdb)
		}},
		&funcMigration{name: "29990201000001_add_column_and_read", up: func(mdb ports.DBAdapter) error {
			executor, err := sqlExecutor(mdb)
			if err != nil {
				return err
			}
			if err := executor.ExecuteMigrationSQL(ports.MigrationSQL{Postgres: []string{"ALTER TABLE mig_tx_test ADD COLUMN extra TEXT"}}); err != nil {
				return err
			}
			return read(mdb)
		}},
	))
	row, err := db.QueryBuilder().Table(migTxTable).First()
	mustNoErr(t, err)
	if row["k"] != "kept" {
		t.Fatalf("migration lost row: %v", row)
	}
	if _, ok := row["extra"]; !ok {
		t.Fatalf("new column missing: %v", row)
	}
}
