package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	payloads := []struct {
		typ byte
		p   []byte
	}{
		{FrameStdout, []byte("hello")},
		{FrameStderr, []byte{0, 1, 2, 255, 0}}, // binary-clean incl. NULs
		{FrameStdout, nil},                     // empty frame
	}
	for _, x := range payloads {
		if err := fw.Write(x.typ, x.p); err != nil {
			t.Fatal(err)
		}
	}
	if err := fw.WriteExit(42); err != nil {
		t.Fatal(err)
	}
	for _, x := range payloads {
		typ, got, err := ReadFrame(&buf)
		if err != nil {
			t.Fatal(err)
		}
		if typ != x.typ || !bytes.Equal(got, x.p) {
			t.Fatalf("frame mismatch: got (%d,%v) want (%d,%v)", typ, got, x.typ, x.p)
		}
	}
	typ, p, err := ReadFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if typ != FrameExit || ExitCode(p) != 42 {
		t.Fatalf("exit frame: typ=%d code=%d", typ, ExitCode(p))
	}
}

func TestReadFrameRejectsOversize(t *testing.T) {
	var hdr [5]byte
	hdr[0] = FrameStdout
	binary.BigEndian.PutUint32(hdr[1:], MaxFrame+1)
	if _, _, err := ReadFrame(bytes.NewReader(hdr[:])); err == nil {
		t.Fatal("accepted oversize frame length")
	}
}

func TestReadFrameTruncated(t *testing.T) {
	// Header promises 10 bytes; only 3 follow.
	var hdr [5]byte
	hdr[0] = FrameStdout
	binary.BigEndian.PutUint32(hdr[1:], 10)
	r := io.MultiReader(bytes.NewReader(hdr[:]), bytes.NewReader([]byte("abc")))
	if _, _, err := ReadFrame(r); err == nil {
		t.Fatal("accepted truncated payload")
	}
}

func TestWriteRejectsOversizePayload(t *testing.T) {
	fw := NewFrameWriter(io.Discard)
	if err := fw.Write(FrameStdout, make([]byte, MaxFrame+1)); err == nil {
		t.Fatal("accepted oversize payload")
	}
}

func TestScore(t *testing.T) {
	if (Envelope{CPUs: 0}).Score() != 1e9 {
		t.Fatal("zero-core machine should sort last")
	}
	a := Envelope{CPUs: 8, Load1: 2}
	b := Envelope{CPUs: 8, Load1: 4}
	if !(a.Score() < b.Score()) {
		t.Fatal("lower load should score better")
	}
}

// FuzzReadFrame ensures ReadFrame never panics and honors the size cap.
func FuzzReadFrame(f *testing.F) {
	f.Add([]byte{1, 0, 0, 0, 3, 'a', 'b', 'c'})
	f.Add([]byte{2, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, p, err := ReadFrame(bytes.NewReader(data))
		if err == nil && len(p) > MaxFrame {
			t.Fatal("returned payload above MaxFrame")
		}
	})
}

// TestFrameWriteZeroAlloc guards the hot-path invariant: steady-state framing
// allocates nothing beyond the caller's payload. CI fails if this regresses.
func TestFrameWriteZeroAlloc(t *testing.T) {
	fw := NewFrameWriter(io.Discard)
	payload := make([]byte, 32*1024)
	got := testing.AllocsPerRun(1000, func() {
		_ = fw.Write(FrameStdout, payload)
	})
	if got != 0 {
		t.Fatalf("frame write allocates %.0f objects/op, want 0", got)
	}
}
