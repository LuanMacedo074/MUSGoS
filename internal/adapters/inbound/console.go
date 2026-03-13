package inbound

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"fsos-server/internal/domain/ports"

	"golang.org/x/crypto/bcrypt"
)

type Console struct {
	db               ports.DBAdapter
	logger           ports.Logger
	reader           io.Reader
	defaultUserLevel int
}

func NewConsole(db ports.DBAdapter, logger ports.Logger, reader io.Reader, defaultUserLevel int) *Console {
	return &Console{
		db:               db,
		logger:           logger,
		reader:           reader,
		defaultUserLevel: defaultUserLevel,
	}
}

func (c *Console) Run() {
	scanner := bufio.NewScanner(c.reader)
	fmt.Print("> ")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print("> ")
			continue
		}

		c.execute(line)
		fmt.Print("> ")
	}
}

func (c *Console) execute(line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	// Join first two words as command (e.g. "create user")
	cmd := parts[0]
	if len(parts) > 1 {
		cmd = parts[0] + " " + parts[1]
	}

	switch strings.ToLower(cmd) {
	case "create user":
		c.createUser(parts[2:])
	case "ban user":
		c.banUser(parts[2:])
	case "revoke ban":
		c.revokeBan(parts[2:])
	case "help":
		c.help()
	case "quit", "exit":
		fmt.Println("Use Ctrl+C to stop the server.")
	default:
		fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmd)
	}
}

func (c *Console) createUser(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: create user <username> <password>")
		return
	}

	username := args[0]
	password := args[1]

	hash, err := hashPassword(password)
	if err != nil {
		fmt.Printf("Error hashing password: %v\n", err)
		return
	}

	if err := c.db.CreateUser(username, hash, c.defaultUserLevel); err != nil {
		fmt.Printf("Error creating user: %v\n", err)
		return
	}

	fmt.Printf("User '%s' created successfully.\n", username)
	c.logger.Info("User created via console", map[string]interface{}{
		"username": username,
	})
}

func (c *Console) banUser(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: ban user <username> [reason]")
		return
	}

	username := args[0]
	reason := "banned via console"
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}

	user, err := c.db.GetUser(username)
	if err != nil {
		fmt.Printf("Error: user '%s' not found.\n", username)
		return
	}

	if err := c.db.CreateBan(&user.ID, nil, reason, nil); err != nil {
		fmt.Printf("Error banning user: %v\n", err)
		return
	}

	fmt.Printf("User '%s' banned. Reason: %s\n", username, reason)
	c.logger.Info("User banned via console", map[string]interface{}{
		"username": username,
		"reason":   reason,
	})
}

func (c *Console) revokeBan(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: revoke ban <username>")
		return
	}

	username := args[0]

	user, err := c.db.GetUser(username)
	if err != nil {
		fmt.Printf("Error: user '%s' not found.\n", username)
		return
	}

	ban, err := c.db.GetActiveBanByUserID(user.ID)
	if err != nil {
		fmt.Printf("Error: no active ban for user '%s'.\n", username)
		return
	}

	if err := c.db.RevokeBan(ban.ID); err != nil {
		fmt.Printf("Error revoking ban: %v\n", err)
		return
	}

	fmt.Printf("Ban revoked for user '%s'.\n", username)
	c.logger.Info("Ban revoked via console", map[string]interface{}{
		"username": username,
	})
}

func (c *Console) help() {
	fmt.Println("Available commands:")
	fmt.Println("  create user <username> <password>  - Create a new user")
	fmt.Println("  ban user <username> [reason]        - Ban a user")
	fmt.Println("  revoke ban <username>               - Revoke active ban for a user")
	fmt.Println("  help                                - Show this help")
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
