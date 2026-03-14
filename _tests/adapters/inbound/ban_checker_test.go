package inbound_test

import (
	"testing"

	"fsos-server/internal/adapters/inbound"
	"fsos-server/internal/domain/ports"

	"fsos-server/_tests/testutil"
)

func TestBanChecker_NilDB(t *testing.T) {
	bc := inbound.NewBanChecker(nil, testutil.NewMockCache())
	if bc.IsIPBanned("1.2.3.4") {
		t.Fatal("expected false when DB is nil")
	}
}

func TestBanChecker_NotBanned(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetActiveBanByIPFunc: func(ip string) (*ports.Ban, error) {
			return nil, ports.ErrBanNotFound
		},
	}
	cache := testutil.NewMockCache()
	bc := inbound.NewBanChecker(db, cache)

	if bc.IsIPBanned("1.2.3.4") {
		t.Fatal("expected false for non-banned IP")
	}

	val, err := cache.Get("ban:ip:1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected cache error: %v", err)
	}
	if string(val) != "0" {
		t.Fatalf("expected cache value '0', got %q", string(val))
	}
}

func TestBanChecker_Banned(t *testing.T) {
	db := &testutil.MockDBAdapter{
		GetActiveBanByIPFunc: func(ip string) (*ports.Ban, error) {
			return &ports.Ban{ID: 1, Reason: "spam"}, nil
		},
	}
	cache := testutil.NewMockCache()
	bc := inbound.NewBanChecker(db, cache)

	if !bc.IsIPBanned("1.2.3.4") {
		t.Fatal("expected true for banned IP")
	}

	val, err := cache.Get("ban:ip:1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected cache error: %v", err)
	}
	if string(val) != "1" {
		t.Fatalf("expected cache value '1', got %q", string(val))
	}
}

func TestBanChecker_CacheHit(t *testing.T) {
	dbCalled := false
	db := &testutil.MockDBAdapter{
		GetActiveBanByIPFunc: func(ip string) (*ports.Ban, error) {
			dbCalled = true
			return nil, ports.ErrBanNotFound
		},
	}
	cache := testutil.NewMockCache()
	cache.Set("ban:ip:1.2.3.4", []byte("1"), 0)

	bc := inbound.NewBanChecker(db, cache)

	if !bc.IsIPBanned("1.2.3.4") {
		t.Fatal("expected true from cached ban")
	}
	if dbCalled {
		t.Fatal("DB should not have been called on cache hit")
	}
}

func TestBanChecker_CacheMiss(t *testing.T) {
	dbCalled := false
	db := &testutil.MockDBAdapter{
		GetActiveBanByIPFunc: func(ip string) (*ports.Ban, error) {
			dbCalled = true
			return &ports.Ban{ID: 1, Reason: "abuse"}, nil
		},
	}
	cache := testutil.NewMockCache()
	bc := inbound.NewBanChecker(db, cache)

	if !bc.IsIPBanned("10.0.0.1") {
		t.Fatal("expected true for banned IP")
	}
	if !dbCalled {
		t.Fatal("DB should have been called on cache miss")
	}

	val, _ := cache.Get("ban:ip:10.0.0.1")
	if string(val) != "1" {
		t.Fatalf("expected cache updated to '1', got %q", string(val))
	}
}
