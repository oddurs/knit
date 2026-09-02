package agent

import (
	"bufio"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/oddurs/knit/internal/keys"
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
	stopping := make(chan struct{})
	ret := make(chan error, 1)
	go func() { ret <- serve(ln, testKey(), stopping) }()

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

// TestRejectsOlderProtocol: a client below the agent's protocol version gets a
// stable `version` error naming the fix instead of a hung or garbled stream.
func TestRejectsOlderProtocol(t *testing.T) {
	addr := serveOne(t, testKey())
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	greeting, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	nonce := strings.Fields(greeting)[1]
	req := proto.Request{V: 1, HMAC: keys.Sign(testKey(), nonce), Op: proto.OpRun, Cmd: []string{"true"}}
	line, _ := json.Marshal(req)
	if _, err := conn.Write(append(line, '\n')); err != nil {
		t.Fatal(err)
	}
	reply, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var env proto.Envelope
	if err := json.Unmarshal([]byte(reply), &env); err != nil {
		t.Fatal(err)
	}
	if env.OK || env.Code != proto.CodeVersion {
		t.Fatalf("got %+v, want code %q", env, proto.CodeVersion)
	}
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
