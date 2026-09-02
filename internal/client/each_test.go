package client

import (
	"bytes"
	"sync"
	"testing"
)

// A line split across two frames must come out as one prefixed line, and a
// final line without a newline must still be emitted on flush.
func TestPrefixerJoinsSplitLines(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	p := &prefixer{mu: &mu, w: &buf, prefix: "[a] "}
	p.Write([]byte("hello, wo"))
	p.Write([]byte("rld\nsecond\nthi"))
	p.Write([]byte("rd"))
	p.flush()
	want := "[a] hello, world\n[a] second\n[a] third\n"
	if got := buf.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
