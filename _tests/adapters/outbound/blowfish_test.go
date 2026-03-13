package outbound_test

import (
	"bytes"
	"testing"

	"fsos-server/internal/adapters/outbound"
)

func newBlowfish(key string) *outbound.Blowfish {
	bf := outbound.NewBlowfish(key)
	bf.SetKey()
	return bf
}

func TestBlowfish_EncryptDecrypt_RoundTrip(t *testing.T) {
	bf := newBlowfish("secretkey")
	original := []byte("Hello, World! This is a test message.")

	encrypted := bf.Encrypt(original)
	decrypted := bf.Decrypt(encrypted)

	if !bytes.Equal(decrypted, original) {
		t.Errorf("round-trip failed:\n  original:  %X\n  decrypted: %X", original, decrypted)
	}
}

func TestBlowfish_DifferentKeys(t *testing.T) {
	bf1 := newBlowfish("key1")
	bf2 := newBlowfish("key2")

	data := []byte("same input data")

	enc1 := bf1.Encrypt(data)
	enc2 := bf2.Encrypt(data)

	if bytes.Equal(enc1, enc2) {
		t.Error("different keys should produce different encrypted output")
	}
}

func TestBlowfish_EmptyInput(t *testing.T) {
	bf := newBlowfish("testkey")

	// Should not panic
	enc := bf.Encrypt([]byte{})
	if len(enc) != 0 {
		t.Errorf("Encrypt(empty) should return empty, got len=%d", len(enc))
	}

	dec := bf.Decrypt([]byte{})
	if len(dec) != 0 {
		t.Errorf("Decrypt(empty) should return empty, got len=%d", len(dec))
	}
}

func TestBlowfish_StateIsolation(t *testing.T) {
	bf := newBlowfish("testkey")
	data := []byte("test data for isolation")

	// Two consecutive encryptions should produce the same result
	enc1 := bf.Encrypt(data)
	enc2 := bf.Encrypt(data)

	if !bytes.Equal(enc1, enc2) {
		t.Errorf("state isolation failed:\n  enc1: %X\n  enc2: %X", enc1, enc2)
	}

	// Two consecutive decryptions should also be consistent
	dec1 := bf.Decrypt(enc1)
	dec2 := bf.Decrypt(enc1)

	if !bytes.Equal(dec1, dec2) {
		t.Errorf("decrypt state isolation failed:\n  dec1: %X\n  dec2: %X", dec1, dec2)
	}
}
