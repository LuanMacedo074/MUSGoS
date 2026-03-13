package lingo_test

import (
	"bytes"
	"testing"

	"fsos-server/internal/domain/types/lingo"
)

func TestLMedia_NewAndToBytes(t *testing.T) {
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	v := lingo.NewLMedia(data)

	if v.GetType() != lingo.VtMedia {
		t.Errorf("GetType() = %d, want %d", v.GetType(), lingo.VtMedia)
	}

	got := v.ToBytes()
	if !bytes.Equal(got, data) {
		t.Errorf("ToBytes() = %X, want %X", got, data)
	}
}
