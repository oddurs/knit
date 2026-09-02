package agent

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/oddurs/connex/internal/proto"
)

// pumpBuf is the reusable copy buffer size for the output pumps.
const pumpBufSize = 64 * 1024

var pumpPool = sync.Pool{New: func() any { b := make([]byte, pumpBufSize); return &b }}

// handleRun spawns the requested command in the agent's home directory and
// streams its stdio. stdin is fed from the connection (half-close propagates
// EOF); stdout/stderr return as typed frames; the exit code is the final frame.
//
// If the client disappears before the process exits (e.g. Ctrl-C closes the
// connection), a frame write fails and we terminate the process group so no
// orphan lingers. This is the v0.1 90% answer to signals; full framing is
// CONNEX2 (CX-EXEC-020). See docs/03-protocol.md.
func handleRun(conn net.Conn, br *bufio.Reader, req proto.Request) {
	if len(req.Cmd) == 0 {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeEmptyCmd, Error: "empty command"})
		return
	}
	c := exec.Command(req.Cmd[0], req.Cmd[1:]...)
	if home, err := os.UserHomeDir(); err == nil {
		c.Dir = home
	}
	// Own process group so we can reap the whole tree on client disconnect.
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := c.StdinPipe()
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeSpawn, Error: err.Error()})
		return
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeSpawn, Error: err.Error()})
		return
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeSpawn, Error: err.Error()})
		return
	}
	if err := c.Start(); err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeSpawn, Error: err.Error()})
		return
	}
	writeEnvelope(conn, proto.Envelope{OK: true})
	log.Printf("run: %v", req.Cmd)

	var killOnce sync.Once
	kill := func() {
		killOnce.Do(func() {
			if c.Process == nil {
				return
			}
			// Signal the whole process group (negative pid).
			_ = syscall.Kill(-c.Process.Pid, syscall.SIGINT)
			time.AfterFunc(2*time.Second, func() {
				_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
			})
		})
	}

	// Feed the remainder of the connection to stdin. A client full-close (Ctrl-C)
	// or half-close both end this copy; we close stdin to propagate EOF.
	go func() {
		_, _ = io.Copy(stdin, br)
		_ = stdin.Close()
	}()

	fw := proto.NewFrameWriter(conn)
	done := make(chan struct{}, 2)
	pump := func(t byte, r io.Reader) {
		bufp := pumpPool.Get().(*[]byte)
		buf := *bufp
		defer pumpPool.Put(bufp)
		for {
			n, rerr := r.Read(buf)
			if n > 0 {
				if werr := fw.Write(t, buf[:n]); werr != nil {
					kill() // client gone; reap the process
					break
				}
			}
			if rerr != nil {
				break
			}
		}
		done <- struct{}{}
	}
	go pump(proto.FrameStdout, stdout)
	go pump(proto.FrameStderr, stderr)
	<-done
	<-done

	code := 0
	if err := c.Wait(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	}
	_ = fw.WriteExit(code)
}
