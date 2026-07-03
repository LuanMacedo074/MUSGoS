//go:build integration

package outbound_test

import (
	"sort"
	"testing"
	"time"
)

func TestIntegrationRedisCache_SetGet(t *testing.T) {
	c := newRedisCache(t)

	mustNoErr(t, c.Set("key1", []byte("value1"), 0))
	got, err := c.Get("key1")
	mustNoErr(t, err)
	if string(got) != "value1" {
		t.Errorf("Get = %q, want %q", got, "value1")
	}

	got, err = c.Get("nonexistent")
	mustNoErr(t, err)
	if got != nil {
		t.Errorf("Get missing = %v, want nil", got)
	}
}

func TestIntegrationRedisCache_DeleteAndExists(t *testing.T) {
	c := newRedisCache(t)

	mustNoErr(t, c.Set("k", []byte("v"), 0))
	exists, err := c.Exists("k")
	mustNoErr(t, err)
	if !exists {
		t.Error("Exists should be true after Set")
	}

	mustNoErr(t, c.Delete("k"))
	exists, err = c.Exists("k")
	mustNoErr(t, err)
	if exists {
		t.Error("Exists should be false after Delete")
	}
}

func TestIntegrationRedisCache_TTLExpiry(t *testing.T) {
	c := newRedisCache(t)

	mustNoErr(t, c.Set("k", []byte("v"), 300*time.Millisecond))
	got, err := c.Get("k")
	mustNoErr(t, err)
	if string(got) != "v" {
		t.Errorf("Get before expiry = %q, want %q", got, "v")
	}

	time.Sleep(500 * time.Millisecond)
	got, err = c.Get("k")
	mustNoErr(t, err)
	if got != nil {
		t.Errorf("Get after expiry = %v, want nil", got)
	}
}

func TestIntegrationRedisCache_SetOps(t *testing.T) {
	c := newRedisCache(t)

	// empty set
	members, err := c.SetMembers("myset")
	mustNoErr(t, err)
	if len(members) != 0 {
		t.Errorf("SetMembers on empty = %v, want []", members)
	}

	mustNoErr(t, c.SetAdd("myset", "a"))
	mustNoErr(t, c.SetAdd("myset", "b"))
	mustNoErr(t, c.SetAdd("myset", "c"))

	members, err = c.SetMembers("myset")
	mustNoErr(t, err)
	sort.Strings(members)
	if len(members) != 3 || members[0] != "a" || members[1] != "b" || members[2] != "c" {
		t.Errorf("SetMembers = %v, want [a b c]", members)
	}

	ok, err := c.SetIsMember("myset", "a")
	mustNoErr(t, err)
	if !ok {
		t.Error("SetIsMember(a) should be true")
	}
	ok, err = c.SetIsMember("myset", "z")
	mustNoErr(t, err)
	if ok {
		t.Error("SetIsMember(z) should be false")
	}

	mustNoErr(t, c.SetRemove("myset", "a"))
	ok, err = c.SetIsMember("myset", "a")
	mustNoErr(t, err)
	if ok {
		t.Error("SetIsMember(a) should be false after SetRemove")
	}
}
