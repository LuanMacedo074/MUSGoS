package outbound

import (
	"testing"
	"time"

	"fsos-server/internal/domain/ports"

	lua "github.com/yuin/gopher-lua"
)

// rowToLuaTable must surface a time.Time (how pgx scans a TIMESTAMPTZ) as an
// RFC3339 string, not nil. Before the fix a datetime column reached Lua as nil on
// Postgres, so a script could neither read a timestamp nor tell a set datetime from
// a NULL one. A NULL still arrives as nil, so nil stays an unambiguous "unset".
func TestRowToLuaTable_TimeColumn(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ts := time.Date(2026, 7, 26, 4, 3, 14, 0, time.UTC)
	tbl := rowToLuaTable(L, ports.QueryResult{
		"deliver_at": ts,
		"claimed_at": nil,
	})

	got := tbl.RawGetString("deliver_at")
	if got.Type() != lua.LTString {
		t.Fatalf("deliver_at type = %s, want string", got.Type())
	}
	if s := lua.LVAsString(got); s != "2026-07-26T04:03:14Z" {
		t.Errorf("deliver_at = %q, want RFC3339 UTC", s)
	}
	if claimed := tbl.RawGetString("claimed_at"); claimed != lua.LNil {
		t.Errorf("claimed_at = %v, want nil (NULL stays unambiguously unset)", claimed)
	}
}
