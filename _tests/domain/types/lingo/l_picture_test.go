package lingo_test

import (
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLPicture_NewAndData(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	v := lingo.NewLPicture(data)
	if v.GetType() != lingo.VtPicture {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtPicture)
	}
	if len(v.Data) != 4 {
		t.Errorf("Data length = %d, want 4", len(v.Data))
	}
}

func TestLPicture_GetBytes_RoundTrip(t *testing.T) {
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x42}
	v := lingo.NewLPicture(data)
	b := v.GetBytes()
	parsed := lingo.FromRawBytes(b, 0)
	pic, ok := parsed.(*lingo.LPicture)
	if !ok {
		t.Fatalf("FromRawBytes returned %T, want *LPicture", parsed)
	}
	if len(pic.Data) != len(data) {
		t.Fatalf("Data length = %d, want %d", len(pic.Data), len(data))
	}
	for i, b := range data {
		if pic.Data[i] != b {
			t.Errorf("Data[%d] = 0x%02X, want 0x%02X", i, pic.Data[i], b)
		}
	}
}

func TestLPicture_String(t *testing.T) {
	v := lingo.NewLPicture([]byte{1, 2, 3})
	got := v.String()
	if got != "<picture 3 bytes>" {
		t.Errorf("String() = %q, want %q", got, "<picture 3 bytes>")
	}
}

func TestLPicture_ToBytes(t *testing.T) {
	data := []byte{0x01, 0x02}
	v := lingo.NewLPicture(data)
	if len(v.ToBytes()) != 2 {
		t.Errorf("ToBytes() length = %d, want 2", len(v.ToBytes()))
	}
}
