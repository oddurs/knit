package proto

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

// FrameWriter serializes typed frames onto a writer. It is safe for use by
// multiple goroutines (the stdout and stderr pumps share one). Each frame is
// written with a single vectored write (header+payload) when the underlying
// writer is a net.Conn, so there is one writev(2) per frame. The header buffer
// and the two-element iovec are held on the writer and reused under the lock,
// so steady-state framing allocates nothing beyond the caller's payload.
type FrameWriter struct {
	mu   sync.Mutex
	w    io.Writer
	hdr  [5]byte
	back [2][]byte
	bufs net.Buffers // reused iovec; kept on the struct so it never escapes
}

// NewFrameWriter wraps w.
func NewFrameWriter(w io.Writer) *FrameWriter { return &FrameWriter{w: w} }

// Write emits one frame of type t carrying p.
func (fw *FrameWriter) Write(t byte, p []byte) error {
	if len(p) > MaxFrame {
		return fmt.Errorf("frame payload too large (%d bytes)", len(p))
	}
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.hdr[0] = t
	binary.BigEndian.PutUint32(fw.hdr[1:], uint32(len(p)))
	if len(p) == 0 {
		_, err := fw.w.Write(fw.hdr[:])
		return err
	}
	// Reuse the struct-held backing array so no slice header escapes to the heap.
	fw.back[0] = fw.hdr[:]
	fw.back[1] = p
	fw.bufs = fw.back[:2]
	_, err := fw.bufs.WriteTo(fw.w) // one writev(2) on a net.Conn
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
