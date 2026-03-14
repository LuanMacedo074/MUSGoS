package outbound_test

import (
	"testing"
	"time"

	"fsos-server/internal/adapters/outbound"
)

func TestMemoryCache_SetGet(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	if err := c.Set("key1", []byte("value1"), 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := c.Get("key1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "value1" {
		t.Errorf("Get = %q, want %q", got, "value1")
	}
}

func TestMemoryCache_GetMissing(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	got, err := c.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("Get = %v, want nil", got)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	c.Set("key1", []byte("value1"), 0)
	c.Delete("key1")

	got, _ := c.Get("key1")
	if got != nil {
		t.Errorf("Get after Delete = %v, want nil", got)
	}
}

func TestMemoryCache_Exists(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	exists, _ := c.Exists("key1")
	if exists {
		t.Error("Exists should be false for missing key")
	}

	c.Set("key1", []byte("value1"), 0)
	exists, _ = c.Exists("key1")
	if !exists {
		t.Error("Exists should be true after Set")
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	c.Set("key1", []byte("value1"), 50*time.Millisecond)

	got, _ := c.Get("key1")
	if string(got) != "value1" {
		t.Errorf("Get before expiry = %q, want %q", got, "value1")
	}

	time.Sleep(60 * time.Millisecond)

	got, _ = c.Get("key1")
	if got != nil {
		t.Errorf("Get after expiry = %v, want nil", got)
	}

	exists, _ := c.Exists("key1")
	if exists {
		t.Error("Exists should be false after expiry")
	}
}

func TestMemoryCache_Overwrite(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	c.Set("key1", []byte("v1"), 0)
	c.Set("key1", []byte("v2"), 0)

	got, _ := c.Get("key1")
	if string(got) != "v2" {
		t.Errorf("Get = %q, want %q", got, "v2")
	}
}

func TestMemoryCache_NoTTL(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	c.Set("key1", []byte("value1"), 0)
	time.Sleep(10 * time.Millisecond)

	got, _ := c.Get("key1")
	if string(got) != "value1" {
		t.Errorf("Get = %q, want %q", got, "value1")
	}
}

func TestMemoryCache_IsolatedCopy(t *testing.T) {
	c := outbound.NewMemoryCache()
	defer c.Close()

	original := []byte("original")
	c.Set("key1", original, 0)
	original[0] = 'X'

	got, _ := c.Get("key1")
	if string(got) != "original" {
		t.Errorf("Get = %q, want %q (mutation leaked)", got, "original")
	}
}
