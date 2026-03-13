package ports

import "fsos-server/internal/domain/types/lingo"

type DBAdapter interface {
	// DBAdmin
	CreateApplication(appName string) error
	DeleteApplication(appName string) error

	// DBApplication (global app data)
	SetApplicationAttribute(appName, attrName string, value lingo.LValue) error
	GetApplicationAttribute(appName, attrName string) (lingo.LValue, error)
	GetApplicationAttributeNames(appName string) ([]string, error)
	DeleteApplicationAttribute(appName, attrName string) error

	// DBPlayer (persistent per userID)
	SetPlayerAttribute(appName, userID, attrName string, value lingo.LValue) error
	GetPlayerAttribute(appName, userID, attrName string) (lingo.LValue, error)
	GetPlayerAttributeNames(appName, userID string) ([]string, error)
	DeletePlayerAttribute(appName, userID, attrName string) error

	// DBUser (session, per clientID)
	SetUserAttribute(clientID, attrName string, value lingo.LValue) error
	GetUserAttribute(clientID, attrName string) (lingo.LValue, error)
	GetUserAttributeNames(clientID string) ([]string, error)
	DeleteUserAttribute(clientID, attrName string) error

	// ExecRaw executes raw SQL — used by migrations for schema changes.
	ExecRaw(sql string) error

	Close() error
}
