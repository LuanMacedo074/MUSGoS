package outbound_test

import (
	"testing"

	"fsos-server/internal/adapters/outbound"
)

// RebindDollar converts ?-style placeholders to Postgres $N placeholders. It
// backs the Postgres dialect's rebinding of the shared SQL, so it must number
// placeholders left to right and must not touch question marks inside quoted
// string literals. (No query in the codebase embeds ? in a literal today; the
// quote handling is a guardrail, per RFC-007.)
func TestRebindDollar(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no placeholders",
			in:   "SELECT name FROM migrations ORDER BY name",
			want: "SELECT name FROM migrations ORDER BY name",
		},
		{
			name: "single placeholder",
			in:   "INSERT INTO applications (name) VALUES (?)",
			want: "INSERT INTO applications (name) VALUES ($1)",
		},
		{
			name: "multiple placeholders numbered left to right",
			in:   "INSERT INTO migrations (name, applied_at) VALUES (?, ?)",
			want: "INSERT INTO migrations (name, applied_at) VALUES ($1, $2)",
		},
		{
			name: "multiline upsert",
			in: `INSERT INTO app_attributes (app_id, attr_name, value_json) VALUES (?, ?, ?)
		ON CONFLICT(app_id, attr_name) DO UPDATE SET value_json=excluded.value_json`,
			want: `INSERT INTO app_attributes (app_id, attr_name, value_json) VALUES ($1, $2, $3)
		ON CONFLICT(app_id, attr_name) DO UPDATE SET value_json=excluded.value_json`,
		},
		{
			name: "question mark inside string literal is not a placeholder",
			in:   "SELECT * FROM users WHERE username = '?' AND user_level = ?",
			want: "SELECT * FROM users WHERE username = '?' AND user_level = $1",
		},
		{
			name: "escaped quote inside literal does not end the literal",
			in:   "UPDATE users SET note = 'what''s this?' WHERE username = ?",
			want: "UPDATE users SET note = 'what''s this?' WHERE username = $1",
		},
		{
			name: "placeholder between two literals",
			in:   "SELECT 'a?' , ? , 'b?' , ?",
			want: "SELECT 'a?' , $1 , 'b?' , $2",
		},
		{
			name: "empty query",
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := outbound.RebindDollar(tc.in)
			if got != tc.want {
				t.Errorf("RebindDollar(%q)\n got: %q\nwant: %q", tc.in, got, tc.want)
			}
		})
	}
}
