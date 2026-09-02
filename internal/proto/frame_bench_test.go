package proto

import (
	"io"
	"testing"
)

// BenchmarkFrameWrite guards the hot-path allocation budget. The payload buffer
// is owned by the caller (pooled in the agent), so steady-state framing should
// not allocate per frame beyond the vectored-write machinery.
func BenchmarkFrameWrite(b *testing.B) {
	fw := NewFrameWriter(io.Discard)
	payload := make([]byte, 32*1024)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := fw.Write(FrameStdout, payload); err != nil {
			b.Fatal(err)
		}
	}
}
