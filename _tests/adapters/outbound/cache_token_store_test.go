package outbound_test

import (
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func newTokenStore(ttl time.Duration) *outbound.CacheTokenStore {
	return outbound.NewCacheTokenStore(outbound.NewMemoryCache(), ttl)
}

func TestCacheTokenStore_IssueThenValidate(t *testing.T) {
	store := newTokenStore(time.Minute)

	token, err := store.Issue(ports.TokenPayload{CharacterName: "Aragorn", UserLevel: 40, AccountID: 7})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if token == "" {
		t.Fatal("Issue returned an empty token")
	}

	payload, err := store.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if payload.CharacterName != "Aragorn" {
		t.Errorf("CharacterName = %q, want Aragorn", payload.CharacterName)
	}
	if payload.UserLevel != 40 {
		t.Errorf("UserLevel = %d, want 40", payload.UserLevel)
	}
	if payload.AccountID != 7 {
		t.Errorf("AccountID = %d, want 7", payload.AccountID)
	}
}

func TestCacheTokenStore_ValidateConsumes(t *testing.T) {
	store := newTokenStore(time.Minute)

	token, _ := store.Issue(ports.TokenPayload{CharacterName: "Legolas", UserLevel: 20})

	if _, err := store.Validate(token); err != nil {
		t.Fatalf("first Validate: %v", err)
	}
	// Single-use: the second validation must fail.
	if _, err := store.Validate(token); err != ports.ErrTokenInvalid {
		t.Errorf("second Validate err = %v, want ErrTokenInvalid", err)
	}
}

func TestCacheTokenStore_UnknownTokenFails(t *testing.T) {
	store := newTokenStore(time.Minute)

	if _, err := store.Validate("deadbeefnotarealtoken"); err != ports.ErrTokenInvalid {
		t.Errorf("Validate(unknown) err = %v, want ErrTokenInvalid", err)
	}
	if _, err := store.Validate(""); err != ports.ErrTokenInvalid {
		t.Errorf("Validate(empty) err = %v, want ErrTokenInvalid", err)
	}
}

func TestCacheTokenStore_TTLExpiry(t *testing.T) {
	store := newTokenStore(20 * time.Millisecond)

	token, _ := store.Issue(ports.TokenPayload{CharacterName: "Gimli", UserLevel: 20})

	time.Sleep(40 * time.Millisecond)

	if _, err := store.Validate(token); err != ports.ErrTokenInvalid {
		t.Errorf("Validate after TTL err = %v, want ErrTokenInvalid", err)
	}
}

func TestCacheTokenStore_TokensAreUnique(t *testing.T) {
	store := newTokenStore(time.Minute)

	a, _ := store.Issue(ports.TokenPayload{CharacterName: "X", UserLevel: 1})
	b, _ := store.Issue(ports.TokenPayload{CharacterName: "X", UserLevel: 1})
	if a == b {
		t.Errorf("two issues produced the same token %q", a)
	}
}
