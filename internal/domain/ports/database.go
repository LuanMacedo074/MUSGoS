package ports

import (
	"errors"
	"time"

	"fsos-server/internal/domain/types/lingo"
)

type User struct {
	ID           int64
	UUID         string
	Username     string
	PasswordHash string
	UserLevel    int
	CreatedAt    time.Time
}

type Ban struct {
	ID        int64
	UUID      string
	UserID    *int64
	IPAddress *string
	Reason    string
	ExpiresAt *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

const DefaultUserLevel = 20

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrBanNotFound       = errors.New("ban not found")
	ErrInvalidCredentials = errors.New("invalid credentials format")
)

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

	// DBUser (authentication)
	CreateUser(username, passwordHash string, userLevel int) error
	GetUser(username string) (*User, error)
	DeleteUser(username string) error
	UpdateUserLevel(username string, level int) error
	UpdateUserPassword(username, passwordHash string) error

	// DBBan
	CreateBan(userID *int64, ipAddress *string, reason string, expiresAt *time.Time) error
	GetActiveBanByUserID(userID int64) (*Ban, error)
	GetActiveBanByIP(ipAddress string) (*Ban, error)
	RevokeBan(banID int64) error

	// Schema operations — used by migrations for schema changes.
	CreateTable(def Table) error
	DropTable(name string) error
	CreateIndex(def Index) error

	Close() error
}
