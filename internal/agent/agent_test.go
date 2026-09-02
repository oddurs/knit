package agent

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/transport"
)

// serveOne starts a loopback listener whose connections are handled by the real
// agent connection handler under the given key, and returns its address.
func serveOne(t *testing.T, key []byte) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	cfg, err := TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go Handle(conn, cfg, key)
		}
	}()
	return ln.Addr().String()
}

func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 7)
	}
	return k
}

func TestRunStreamsStdoutStderrAndExit(t *testing.T) {
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(),
		proto.Request{Op: proto.OpRun, Cmd: []string{"sh", "-c", "printf OUT; printf ERR 1>&2; exit 5"}},
		2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	// V2: no stdin, so send an explicit stdin-EOF frame.
	if err := proto.NewFrameWriter(sess.Conn).Write(proto.FrameStdinEOF, nil); err != nil {
		t.Fatal(err)
	}

	var out, errb strings.Builder
	code := -1
	for {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatalf("read frame: %v", err)
		}
		switch typ {
		case proto.FrameStdout:
			out.Write(p)
		case proto.FrameStderr:
			errb.Write(p)
		case proto.FrameExit:
			code = proto.ExitCode(p)
		}
		if code >= 0 {
			break
		}
	}
	if out.String() != "OUT" || errb.String() != "ERR" || code != 5 {
		t.Fatalf("out=%q err=%q code=%d", out.String(), errb.String(), code)
	}
}

func TestRunStdinRoundTrip(t *testing.T) {
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(),
		proto.Request{Op: proto.OpRun, Cmd: []string{"cat"}}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()

	go func() {
		fw := proto.NewFrameWriter(sess.Conn)
		fw.Write(proto.FrameStdin, []byte("roundtrip payload\n"))
		fw.Write(proto.FrameStdinEOF, nil)
	}()

	var out strings.Builder
	for {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatalf("read frame: %v", err)
		}
		if typ == proto.FrameStdout {
			out.Write(p)
		}
		if typ == proto.FrameExit {
			break
		}
	}
	if out.String() != "roundtrip payload\n" {
		t.Fatalf("stdin not echoed: got %q", out.String())
	}
}

func TestInfoOp(t *testing.T) {
	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(), proto.Request{Op: proto.OpInfo}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	if !sess.Reply.OK || sess.Reply.CPUs < 1 || sess.Reply.Name == "" {
		t.Fatalf("implausible info reply: %+v", sess.Reply)
	}
}

func TestUnauthorized(t *testing.T) {
	addr := serveOne(t, testKey())
	wrong := make([]byte, 32) // all zeros != testKey
	_, err := transport.Open(addr, wrong, proto.Request{Op: proto.OpInfo}, 2*time.Second)
	re, ok := err.(*transport.ReplyError)
	if !ok || re.Code != proto.CodeUnauthorized {
		t.Fatalf("expected unauthorized reply error, got %v", err)
	}
}

func TestEmptyCommand(t *testing.T) {
	addr := serveOne(t, testKey())
	_, err := transport.Open(addr, testKey(), proto.Request{Op: proto.OpRun}, 2*time.Second)
	re, ok := err.(*transport.ReplyError)
	if !ok || re.Code != proto.CodeEmptyCmd {
		t.Fatalf("expected empty_cmd reply error, got %v", err)
	}
}
