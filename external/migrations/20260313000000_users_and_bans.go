package migrations

import "fsos-server/internal/domain/ports"

func init() {
	Register(&usersAndBans{})
}

type usersAndBans struct{}

func (m *usersAndBans) Name() string { return "20260313000000_users_and_bans" }

func (m *usersAndBans) Up(db ports.DBAdapter) error {
	return db.ExecRaw(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			salt TEXT NOT NULL,
			user_level INTEGER NOT NULL DEFAULT 20,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS bans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			ip_address TEXT,
			reason TEXT NOT NULL DEFAULT '',
			expires_at DATETIME,
			revoked_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CHECK (user_id IS NOT NULL OR ip_address IS NOT NULL)
		);

		CREATE INDEX IF NOT EXISTS idx_bans_user_id ON bans(user_id);
		CREATE INDEX IF NOT EXISTS idx_bans_ip_address ON bans(ip_address);
		CREATE INDEX IF NOT EXISTS idx_bans_expires_at ON bans(expires_at);
	`)
}

func (m *usersAndBans) Down(db ports.DBAdapter) error {
	return db.ExecRaw(`
		DROP TABLE IF EXISTS bans;
		DROP TABLE IF EXISTS users;
	`)
}
