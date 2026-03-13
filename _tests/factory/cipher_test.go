package factory_test

import (
	"testing"

	"fsos-server/internal/factory"
)

func TestNewCipher_Blowfish(t *testing.T) {
	cipher, err := factory.NewCipher("blowfish", "testkey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cipher == nil {
		t.Fatal("cipher should not be nil")
	}
}

func TestNewCipher_Unknown(t *testing.T) {
	_, err := factory.NewCipher("aes", "testkey")
	if err == nil {
		t.Error("expected error for unknown cipher type")
	}
}
