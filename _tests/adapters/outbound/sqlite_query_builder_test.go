package outbound_test

import (
	"testing"

	"fsos-server/internal/domain/ports"
)

func TestQueryBuilder_InsertAndFirst(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	// Create a custom table for testing
	db.CreateApplication("testApp")

	// Use the applications table which exists from migrations
	row, err := qb.Table("applications").Where("name", "testApp").First()
	if err != nil {
		t.Fatalf("first error: %v", err)
	}
	if row == nil {
		t.Fatal("expected a row, got nil")
	}
	if row["name"] != "testApp" {
		t.Errorf("name = %v, want %q", row["name"], "testApp")
	}
}

func TestQueryBuilder_Get(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	db.CreateApplication("app1")
	db.CreateApplication("app2")

	rows, err := qb.Table("applications").Get()
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 rows, got %d", len(rows))
	}
}

func TestQueryBuilder_Count(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	db.CreateApplication("countApp1")
	db.CreateApplication("countApp2")

	count, err := qb.Table("applications").Count()
	if err != nil {
		t.Fatalf("count error: %v", err)
	}
	if count < 2 {
		t.Errorf("count = %d, want >= 2", count)
	}
}

func TestQueryBuilder_Update(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	db.CreateUser("updateuser", "hash", 20)

	affected, err := qb.Table("users").Where("username", "updateuser").Update(map[string]interface{}{
		"user_level": 80,
	})
	if err != nil {
		t.Fatalf("update error: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected = %d, want 1", affected)
	}

	row, err := qb.Table("users").Where("username", "updateuser").First()
	if err != nil {
		t.Fatalf("first error: %v", err)
	}
	if row == nil {
		t.Fatal("expected a row after update")
	}
	level, ok := row["user_level"].(int64)
	if !ok {
		t.Fatalf("user_level type = %T, want int64", row["user_level"])
	}
	if level != 80 {
		t.Errorf("user_level = %d, want 80", level)
	}
}

func TestQueryBuilder_Increment(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	db.CreateUser("incuser", "hash", 20)

	// SET user_level = user_level + 10 in one statement (atomic; no read-modify-write).
	affected, err := qb.Table("users").Where("username", "incuser").Increment("user_level", 10)
	if err != nil {
		t.Fatalf("increment error: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected = %d, want 1", affected)
	}

	// A negative delta subtracts.
	if _, err := qb.Table("users").Where("username", "incuser").Increment("user_level", -5); err != nil {
		t.Fatalf("increment (negative) error: %v", err)
	}

	row, err := qb.Table("users").Where("username", "incuser").First()
	if err != nil {
		t.Fatalf("first error: %v", err)
	}
	if level, _ := row["user_level"].(int64); level != 25 {
		t.Errorf("user_level = %v, want 25 (20 +10 -5)", row["user_level"])
	}
}

func TestQueryBuilder_Delete(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	db.CreateApplication("deleteMe")

	affected, err := qb.Table("applications").Where("name", "deleteMe").Delete()
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected = %d, want 1", affected)
	}

	row, err := qb.Table("applications").Where("name", "deleteMe").First()
	if err != nil {
		t.Fatalf("first error: %v", err)
	}
	if row != nil {
		t.Error("expected nil after delete")
	}
}

func TestQueryBuilder_WhereChaining(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	db.CreateUser("chainuser", "hash", 50)

	row, err := qb.Table("users").Where("username", "chainuser").Where("user_level", 50).First()
	if err != nil {
		t.Fatalf("first error: %v", err)
	}
	if row == nil {
		t.Fatal("expected a row with chained where")
	}
}

func TestQueryBuilder_FirstNotFound(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	row, err := qb.Table("applications").Where("name", "nonexistent").First()
	if err != nil {
		t.Fatalf("first error: %v", err)
	}
	if row != nil {
		t.Error("expected nil for nonexistent row")
	}
}

func TestQueryBuilder_InvalidTableName(t *testing.T) {
	db := newTestDB(t)
	qb := db.QueryBuilder()

	_, err := qb.Table("bad table!").Count()
	if err == nil {
		t.Error("expected error for invalid table name")
	}
}

// predicateDB is the slice of a storage adapter the predicate assertions need,
// so the same checks run against both SQL dialects.
type predicateDB interface {
	ports.DBAdapter
	QueryBuilder() ports.QueryBuilder
}

func TestQueryBuilder_WhereLessThanFiltersBelowBound(t *testing.T) {
	assertWhereLessThanFiltersBelowBound(t, newTestDB(t))
}

func TestQueryBuilder_WhereNilMatchesNull(t *testing.T) {
	assertWhereNilMatchesNull(t, newTestDB(t))
}

func assertWhereLessThanFiltersBelowBound(t *testing.T, db predicateDB) {
	t.Helper()
	mustNoErr(t, db.CreateUser("novice", "hash", 5))
	mustNoErr(t, db.CreateUser("adept", "hash", 50))
	mustNoErr(t, db.CreateUser("master", "hash", 100))

	query := db.QueryBuilder().Table("users")
	rows, err := query.WhereLessThan("user_level", 50).Get()
	mustNoErr(t, err)
	if len(rows) != 1 || rows[0]["username"] != "novice" {
		t.Fatalf("rows = %v, want only novice (level 5 < 50)", rows)
	}

	count, err := query.WhereLessThan("user_level", 100).Where("username", "adept").Count()
	mustNoErr(t, err)
	if count != 1 {
		t.Fatalf("less-than combined with equality: count = %d, want 1", count)
	}

	total, err := query.Count()
	mustNoErr(t, err)
	if total != 3 {
		t.Fatalf("derived predicate mutated the original query: count = %d, want 3", total)
	}

	if _, err := query.WhereLessThan("user_level OR 1=1", 50).Get(); err == nil {
		t.Fatal("less-than accepted an invalid column")
	}
}

func assertWhereNilMatchesNull(t *testing.T, db predicateDB) {
	t.Helper()
	activeIP, revokedIP := "10.0.0.1", "10.0.0.2"
	mustNoErr(t, db.CreateBan(nil, &activeIP, "active", nil))
	mustNoErr(t, db.CreateBan(nil, &revokedIP, "revoked", nil))

	query := db.QueryBuilder().Table("bans")
	revoked, err := query.Where("reason", "revoked").First()
	mustNoErr(t, err)
	mustNoErr(t, db.RevokeBan(revoked["id"].(int64)))

	rows, err := query.Where("revoked_at", nil).Get()
	mustNoErr(t, err)
	if len(rows) != 1 || rows[0]["reason"] != "active" {
		t.Fatalf("rows = %v, want only the unrevoked ban", rows)
	}

	affected, err := query.Where("revoked_at", nil).Where("ip_address", activeIP).Update(map[string]interface{}{"reason": "renamed"})
	mustNoErr(t, err)
	if affected != 1 {
		t.Fatalf("IS NULL predicate mixed with a bound one: affected = %d, want 1", affected)
	}

	deleted, err := query.Where("revoked_at", nil).Delete()
	mustNoErr(t, err)
	if deleted != 1 {
		t.Fatalf("delete with IS NULL predicate: affected = %d, want 1", deleted)
	}
	remaining, err := query.Count()
	mustNoErr(t, err)
	if remaining != 1 {
		t.Fatalf("remaining bans = %d, want the revoked one", remaining)
	}
}
