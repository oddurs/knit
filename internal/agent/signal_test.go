package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/transport"
)

// TestSignalForwardedV2 verifies KN-EXEC-020: a signal frame from the client
// delivers the actual signal (here SIGTERM) to the remote process, which can
// trap it and exit with its own code — not just be force-killed.
func TestSignalForwardedV2(t *testing.T) {
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(),
		proto.Request{Op: proto.OpRun, Cmd: []string{"sh", "-c", `trap 'printf GOTTERM; exit 42' TERM; sleep 30`}},
		2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	_ = sess.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Give the shell a moment to install its trap, then send SIGTERM (15).
	time.Sleep(300 * time.Millisecond)
	fw := proto.NewFrameWriter(sess.Conn)
	if err := fw.Write(proto.FrameSignal, []byte{byte(15)}); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code := -1
	for code < 0 {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatalf("read frame: %v (out so far: %q)", err, out.String())
		}
		switch typ {
		case proto.FrameStdout:
			out.Write(p)
		case proto.FrameExit:
			code = proto.ExitCode(p)
		}
	}
	if !strings.Contains(out.String(), "GOTTERM") || code != 42 {
		t.Fatalf("signal not trapped: out=%q code=%d", out.String(), code)
	}
}
