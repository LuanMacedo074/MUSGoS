package migrations

import "fsos-server/internal/domain/ports"

func init() {
	Register(&initialSchema{})
}

type initialSchema struct{}

func (m *initialSchema) Name() string { return "00000000000000_initial_schema" }

func (m *initialSchema) Up(db ports.DBAdapter) error {
	if err := db.CreateTable(ports.Table{
		Name: "applications",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.Col("name", ports.ColText).NotNull().Unique(),
		},
	}); err != nil {
		return err
	}

	if err := db.CreateTable(ports.Table{
		Name: "application_attributes",
		Columns: []ports.Column{
			ports.Col("app_id", ports.ColInteger).NotNull(),
			ports.Col("attr_name", ports.ColText).NotNull(),
			ports.Col("value_json", ports.ColText).NotNull(),
		},
		PrimaryKeys: []string{"app_id", "attr_name"},
		ForeignKeys: []ports.ForeignKey{
			{Column: "app_id", RefTable: "applications", RefCol: "id", OnDelete: "CASCADE"},
		},
	}); err != nil {
		return err
	}

	if err := db.CreateTable(ports.Table{
		Name: "player_attributes",
		Columns: []ports.Column{
			ports.Col("app_id", ports.ColInteger).NotNull(),
			ports.Col("user_id", ports.ColText).NotNull(),
			ports.Col("attr_name", ports.ColText).NotNull(),
			ports.Col("value_json", ports.ColText).NotNull(),
		},
		PrimaryKeys: []string{"app_id", "user_id", "attr_name"},
		ForeignKeys: []ports.ForeignKey{
			{Column: "app_id", RefTable: "applications", RefCol: "id", OnDelete: "CASCADE"},
		},
	}); err != nil {
		return err
	}

	if err := db.CreateTable(ports.Table{
		Name: "users",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.UUID("uuid"),
			ports.Col("username", ports.ColText).NotNull().Unique(),
			ports.Col("password_hash", ports.ColText).NotNull(),
			ports.Col("user_level", ports.ColInteger).NotNull().Default(ports.DefaultUserLevel),
			ports.Col("created_at", ports.ColDatetime).NotNull().DefaultNow(),
		},
	}); err != nil {
		return err
	}

	if err := db.CreateTable(ports.Table{
		Name: "bans",
		Columns: []ports.Column{
			ports.PrimaryKey("id"),
			ports.UUID("uuid"),
			ports.Col("user_id", ports.ColInteger),
			ports.Col("ip_address", ports.ColText),
			ports.Col("reason", ports.ColText).NotNull().Default(""),
			ports.Col("expires_at", ports.ColDatetime),
			ports.Col("revoked_at", ports.ColDatetime),
			ports.Col("created_at", ports.ColDatetime).NotNull().DefaultNow(),
		},
		ForeignKeys: []ports.ForeignKey{
			{Column: "user_id", RefTable: "users", RefCol: "id", OnDelete: "CASCADE"},
		},
		RequireOneOf: []string{"user_id", "ip_address"},
	}); err != nil {
		return err
	}

	for _, idx := range []ports.Index{
		{Name: "idx_bans_user_id", Table: "bans", Columns: []string{"user_id"}},
		{Name: "idx_bans_ip_address", Table: "bans", Columns: []string{"ip_address"}},
		{Name: "idx_bans_expires_at", Table: "bans", Columns: []string{"expires_at"}},
	} {
		if err := db.CreateIndex(idx); err != nil {
			return err
		}
	}

	return nil
}

func (m *initialSchema) Down(db ports.DBAdapter) error {
	for _, t := range []string{"bans", "player_attributes", "application_attributes", "users", "applications"} {
		if err := db.DropTable(t); err != nil {
			return err
		}
	}
	return nil
}
