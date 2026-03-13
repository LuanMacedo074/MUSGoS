package migrations

import "fsos-server/internal/domain/ports"

func init() {
	Register(&initialSchema{})
}

type initialSchema struct{}

func (m *initialSchema) Name() string { return "00000000000000_initial_schema" }

func (m *initialSchema) Up(db ports.DBAdapter) error {
	return db.ExecRaw(`
		CREATE TABLE IF NOT EXISTS applications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);

		CREATE TABLE IF NOT EXISTS application_attributes (
			app_id INTEGER NOT NULL,
			attr_name TEXT NOT NULL,
			value_json TEXT NOT NULL,
			PRIMARY KEY (app_id, attr_name),
			FOREIGN KEY (app_id) REFERENCES applications(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS player_attributes (
			app_id INTEGER NOT NULL,
			user_id TEXT NOT NULL,
			attr_name TEXT NOT NULL,
			value_json TEXT NOT NULL,
			PRIMARY KEY (app_id, user_id, attr_name),
			FOREIGN KEY (app_id) REFERENCES applications(id) ON DELETE CASCADE
		);
	`)
}

func (m *initialSchema) Down(db ports.DBAdapter) error {
	return db.ExecRaw(`
		DROP TABLE IF EXISTS player_attributes;
		DROP TABLE IF EXISTS application_attributes;
		DROP TABLE IF EXISTS applications;
	`)
}
