package outbound_test

import (
	"testing"
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
