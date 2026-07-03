package factory

import (
	"fmt"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
	"fsos-server/internal/domain/services"
)

type DatabaseResult struct {
	Adapter         ports.DBAdapter
	MigrationRunner *services.MigrationRunner
	QueryBuilder    ports.QueryBuilder
}

// NewDatabase builds the database adapter for dbType. connStr is a filesystem
// path for sqlite and a connection DSN for postgres.
func NewDatabase(dbType, connStr string, migrations []ports.Migration) (*DatabaseResult, error) {
	switch dbType {
	case "sqlite":
		db, err := outbound.NewSQLiteDB(connStr)
		if err != nil {
			return nil, err
		}

		runner := services.NewMigrationRunner(db, db, migrations)

		return &DatabaseResult{
			Adapter:         db,
			MigrationRunner: runner,
			QueryBuilder:    db.QueryBuilder(),
		}, nil
	case "postgres", "postgresql":
		db, err := outbound.NewPostgresDB(connStr)
		if err != nil {
			return nil, err
		}

		runner := services.NewMigrationRunner(db, db, migrations)

		return &DatabaseResult{
			Adapter:         db,
			MigrationRunner: runner,
			QueryBuilder:    db.QueryBuilder(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
