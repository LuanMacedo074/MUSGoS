package inbound

import (
	"encoding/binary"
	"testing"

	"fsos-server/internal/domain/types/smus"
)

// buildFrame wraps payload in a MUS envelope: [0x72 0x00][size:uint32][payload].
func buildFrame(payload []byte) []byte {
	b := make([]byte, musFrameHeaderLen+len(payload))
	b[0] = smus.MUSHeader[0]
	b[1] = smus.MUSHeader[1]
	binary.BigEndian.PutUint32(b[2:6], uint32(len(payload)))
	copy(b[musFrameHeaderLen:], payload)
	return b
}

// A frame arriving in pieces must be reported incomplete until fully present,
// then extracted whole (TCP splitting, H4).
func TestNextFrame_SplitAcrossReads(t *testing.T) {
	full := buildFrame([]byte("hello"))

	if _, _, ok, err := nextFrame(full[:4], 1024); err != nil || ok {
		t.Fatalf("partial frame: got ok=%v err=%v, want ok=false err=nil", ok, err)
	}

	frame, rest, ok, err := nextFrame(full, 1024)
	if err != nil || !ok {
		t.Fatalf("full frame: got ok=%v err=%v, want ok=true err=nil", ok, err)
	}
	if string(frame) != string(full) {
		t.Errorf("frame mismatch: got %q", frame)
	}
	if len(rest) != 0 {
		t.Errorf("expected no leftover, got %d bytes", len(rest))
	}
}

// Two frames coalesced into one Read must be split into two (TCP coalescing, H4).
func TestNextFrame_Coalesced(t *testing.T) {
	buf := append(buildFrame([]byte("aa")), buildFrame([]byte("bbbb"))...)

	f1, rest, ok, err := nextFrame(buf, 1024)
	if err != nil || !ok || string(f1) != string(buildFrame([]byte("aa"))) {
		t.Fatalf("first frame: ok=%v err=%v", ok, err)
	}
	f2, rest2, ok, err := nextFrame(rest, 1024)
	if err != nil || !ok || string(f2) != string(buildFrame([]byte("bbbb"))) {
		t.Fatalf("second frame: ok=%v err=%v", ok, err)
	}
	if len(rest2) != 0 {
		t.Errorf("expected empty remainder, got %d bytes", len(rest2))
	}
}

func TestNextFrame_BadHeader(t *testing.T) {
	buf := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x41}
	if _, _, _, err := nextFrame(buf, 1024); err == nil {
		t.Fatal("expected protocol error for bad header, got nil")
	}
}

func TestNextFrame_OversizeRejected(t *testing.T) {
	buf := make([]byte, musFrameHeaderLen)
	buf[0], buf[1] = smus.MUSHeader[0], smus.MUSHeader[1]
	binary.BigEndian.PutUint32(buf[2:6], 1_000_000)
	if _, _, _, err := nextFrame(buf, 1024); err == nil {
		t.Fatal("expected oversize error, got nil")
	}
}
