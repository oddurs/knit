package agent

import (
	"net"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/transport"
)

// TestAbortReapsRemoteProcess verifies KN-EXEC-010: when the client aborts the
// connection (as Ctrl-C does, via an RST), the agent reaps the remote process
// group so nothing is orphaned — even a process producing no further output.
func TestAbortReapsRemoteProcess(t *testing.T) {
	addr := serveOne(t, testKey())
	// The command prints its own PID, then sleeps silently for a long time.
	sess, err := transport.Open(addr, testKey(),
		proto.Request{Op: proto.OpRun, Cmd: []string{"sh", "-c", "echo $$; exec sleep 60"}},
		2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Read the PID line from the first stdout frame.
	var pid int
	deadline := time.Now().Add(3 * time.Second)
	for pid == 0 && time.Now().Before(deadline) {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatalf("reading pid frame: %v", err)
		}
		if typ == proto.FrameStdout {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(p)))
		}
	}
	if pid == 0 {
		t.Fatal("did not learn remote pid")
	}
	// Process should be alive now.
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("remote process not running: %v", err)
	}

	// Abort like Ctrl-C: RST the connection.
	if tc, ok := sess.Conn.(*net.TCPConn); ok {
		_ = tc.SetLinger(0)
	}
	sess.Conn.Close()

	// Within a short window the process (and its group) must be gone.
	gone := false
	for i := 0; i < 50; i++ {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			gone = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !gone {
		// Clean up the leaked process so the test machine isn't left with it.
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatal("remote process was orphaned after client abort")
	}
}
