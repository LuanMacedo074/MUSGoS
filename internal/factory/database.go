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

func NewDatabase(dbType, dbPath string, migrations []ports.Migration) (*DatabaseResult, error) {
	switch dbType {
	case "sqlite":
		db, err := outbound.NewSQLiteDB(dbPath)
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
