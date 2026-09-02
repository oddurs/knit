package proto

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

// coalesce is the largest payload the writer joins with its header into a
// single write; it matches the pump buffer so steady-state streaming is one
// write per frame. Larger payloads take two writes.
const coalesce = 64 * 1024

// FrameWriter serializes typed frames onto a writer. It is safe for use by
// multiple goroutines (the stdout and stderr pumps share one). Header and
// payload are joined in a buffer held on the writer, so each frame is one
// write and, over TLS, one record sequence, and steady-state framing allocates
// nothing beyond the caller's payload.
type FrameWriter struct {
	mu  sync.Mutex
	w   io.Writer
	buf []byte
}

// NewFrameWriter wraps w.
func NewFrameWriter(w io.Writer) *FrameWriter {
	return &FrameWriter{w: w, buf: make([]byte, 5+coalesce)}
}

// Write emits one frame of type t carrying p.
func (fw *FrameWriter) Write(t byte, p []byte) error {
	if len(p) > MaxFrame {
		return fmt.Errorf("frame payload too large (%d bytes)", len(p))
	}
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.buf[0] = t
	binary.BigEndian.PutUint32(fw.buf[1:5], uint32(len(p)))
	if len(p) <= coalesce {
		n := 5 + copy(fw.buf[5:], p)
		_, err := fw.w.Write(fw.buf[:n])
		return err
	}
	if _, err := fw.w.Write(fw.buf[:5]); err != nil {
		return err
	}
	_, err := fw.w.Write(p)
	return err
}

// WriteExit emits the terminal exit frame carrying a 4-byte big-endian code.
func (fw *FrameWriter) WriteExit(code int) error {
	var p [4]byte
	binary.BigEndian.PutUint32(p[:], uint32(code))
	return fw.Write(FrameExit, p[:])
}

// ReadFrame reads one frame. It rejects a declared length above MaxFrame
// before allocating, so a malformed peer cannot exhaust memory.
func ReadFrame(r io.Reader) (t byte, payload []byte, err error) {
	var hdr [5]byte
	if _, err = io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	n := binary.BigEndian.Uint32(hdr[1:])
	if n > MaxFrame {
		return 0, nil, fmt.Errorf("frame too large (%d bytes)", n)
	}
	p := make([]byte, n)
	if _, err = io.ReadFull(r, p); err != nil {
		return 0, nil, err
	}
	return hdr[0], p, nil
}

// ExitCode decodes an exit-frame payload.
func ExitCode(payload []byte) int {
	if len(payload) < 4 {
		return 1
	}
	return int(int32(binary.BigEndian.Uint32(payload)))
}
