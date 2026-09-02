package agent

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/transport"
)

// TestServeReturnsOnStop guards against the accept loop spinning after the
// listener is closed for shutdown: `knit down` must end the process.
func TestServeReturnsOnStop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := TLSConfig()
	stopping := make(chan struct{})
	ret := make(chan error, 1)
	go func() { ret <- serve(ln, cfg, func() []byte { return testKey() }, stopping) }()

	close(stopping)
	_ = ln.Close()
	select {
	case err := <-ret:
		if err != nil {
			t.Fatalf("serve returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after the listener was closed for shutdown")
	}
}

// TestAbortAfterStdinEOFReapsRemoteProcess covers the `knit each` shape: the
// client has already sent stdin-EOF, produces nothing more, and then vanishes.
// The silent remote process must still be reaped.
func TestAbortAfterStdinEOFReapsRemoteProcess(t *testing.T) {
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(),
		proto.Request{Op: proto.OpRun, Cmd: []string{"sh", "-c", "echo $$; exec sleep 60"}},
		2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := proto.NewFrameWriter(sess.Conn).Write(proto.FrameStdinEOF, nil); err != nil {
		t.Fatal(err)
	}
	pid := readPid(t, sess)

	if tc, ok := sess.Conn.(*net.TCPConn); ok {
		_ = tc.SetLinger(0)
	}
	sess.Conn.Close()

	for i := 0; i < 50; i++ {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatal("remote process was orphaned after client abort following stdin-EOF")
}

// TestSignalDeathExitCode: a remote process that dies from a signal reports
// 128+signal, exactly as a local shell would, not -1.
func TestSignalDeathExitCode(t *testing.T) {
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(),
		proto.Request{Op: proto.OpRun, Cmd: []string{"sleep", "30"}}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	time.Sleep(200 * time.Millisecond)
	if err := proto.NewFrameWriter(sess.Conn).Write(proto.FrameSignal, []byte{byte(syscall.SIGTERM)}); err != nil {
		t.Fatal(err)
	}
	_ = sess.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatal(err)
		}
		if typ == proto.FrameExit {
			if got := proto.ExitCode(p); got != 143 {
				t.Fatalf("exit code %d, want 143 (128+SIGTERM)", got)
			}
			return
		}
	}
}

// readPid reads the pid the test command prints as its first stdout line.
func readPid(t *testing.T, sess *transport.Session) int {
	t.Helper()
	_ = sess.Conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	defer sess.Conn.SetReadDeadline(time.Time{})
	for {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatalf("reading pid frame: %v", err)
		}
		if typ == proto.FrameStdout {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(p))); err == nil && pid > 0 {
				if err := syscall.Kill(pid, 0); err != nil {
					t.Fatalf("remote process not running: %v", err)
				}
				return pid
			}
		}
	}
}

// TestRankEnvReachesCommand: a request carrying hosts/rank runs the command
// with the rank environment and a readable MLX hostfile (KN-AI-030).
func TestRankEnvReachesCommand(t *testing.T) {
	t.Setenv("KNIT_HOME", t.TempDir())
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(), proto.Request{
		Op: proto.OpRun, Hosts: []string{"10.0.0.1", "10.0.0.2"}, Rank: 1,
		Cmd: []string{"sh", "-c", `echo "$KNIT_RANK/$KNIT_NNODES/$KNIT_MASTER/$MLX_RANK"; cat "$MLX_HOSTFILE"`},
	}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	_ = proto.NewFrameWriter(sess.Conn).Write(proto.FrameStdinEOF, nil)
	_ = sess.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var out strings.Builder
	for {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatal(err)
		}
		if typ == proto.FrameStdout {
			out.Write(p)
		}
		if typ == proto.FrameExit {
			break
		}
	}
	got := out.String()
	if !strings.HasPrefix(got, "1/2/10.0.0.1/1\n") || !strings.Contains(got, `"ips":["10.0.0.2"]`) {
		t.Fatalf("unexpected output:\n%s", got)
	}
	if files, _ := filepath.Glob(filepath.Join(os.Getenv("KNIT_HOME"), "hostfile-*.json")); len(files) != 0 {
		t.Fatalf("hostfile not cleaned up: %v", files)
	}
}

// TestConnectionReuse: an info op followed by a run on the same connection is
// served (KN-XPORT-050), and the run's output and exit code are correct.
func TestConnectionReuse(t *testing.T) {
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(), proto.Request{Op: proto.OpInfo}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	if sess.Reply.CPUs < 1 {
		t.Fatalf("info reply not received: %+v", sess.Reply)
	}
	// Reuse the same connection for a run.
	if err := sess.Do(proto.Request{Op: proto.OpRun, Cmd: []string{"sh", "-c", "printf REUSED; exit 4"}}); err != nil {
		t.Fatal(err)
	}
	_ = proto.NewFrameWriter(sess.Conn).Write(proto.FrameStdinEOF, nil)
	var out strings.Builder
	code := -1
	_ = sess.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for code < 0 {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		switch typ {
		case proto.FrameStdout:
			out.Write(p)
		case proto.FrameExit:
			code = proto.ExitCode(p)
		}
	}
	if out.String() != "REUSED" || code != 4 {
		t.Fatalf("reused run wrong: out=%q code=%d", out.String(), code)
	}
}

// TestDialTunnel: a dial op splices raw bytes to a target the agent can reach,
// which is what knit proxy rides on.
func TestDialTunnel(t *testing.T) {
	// An echo server stands in for "the network the peer can reach".
	echo, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		for {
			c, err := echo.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); io.Copy(c, c) }()
		}
	}()

	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(), proto.Request{Op: proto.OpDial, Host: echo.Addr().String()}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	_ = sess.Conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := sess.Conn.Write([]byte("ping\n")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(sess.Conn, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "ping\n" {
		t.Fatalf("tunnel echoed %q", buf)
	}
}
