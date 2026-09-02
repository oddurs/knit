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

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/treesync"
)

const pumpBufSize = 64 * 1024

var pumpPool = sync.Pool{New: func() any { b := make([]byte, pumpBufSize); return &b }}

// handleRun spawns the requested command in the agent's home directory and
// streams its stdio. The command runs in its own process group so the whole tree
// can be reaped.
//
// Two client->server conventions are supported, chosen by the client's protocol
// version (req.V):
//
//   - V2 (KNIT2): the client sends framed input — stdin chunks, an explicit
//     stdin-EOF, and signal frames — so Ctrl-C forwards the real signal
//     (SIGINT/SIGTERM) to the remote process and a piped stdin's EOF is
//     unambiguous. A dropped connection still reaps the group.
//   - V<=1 (legacy): the client sends raw stdin and half-closes on EOF; an
//     abnormal close reaps the group. Signals cannot be distinguished.
func handleRun(conn net.Conn, br *bufio.Reader, req proto.Request) {
	if len(req.Cmd) == 0 {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeEmptyCmd, Error: "empty command"})
		return
	}
	if req.Dir {
		handleRunDir(conn, br, req)
		return
	}
	c := exec.Command(req.Cmd[0], req.Cmd[1:]...)
	if home, err := os.UserHomeDir(); err == nil {
		c.Dir = home
	}
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

	pid := c.Process.Pid
	var killOnce sync.Once
	kill := func() {
		killOnce.Do(func() {
			_ = syscall.Kill(-pid, syscall.SIGINT)
			time.AfterFunc(2*time.Second, func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
		})
	}
	signalGroup := func(sig syscall.Signal) { _ = syscall.Kill(-pid, sig) }

	if req.V >= 2 {
		go readClientFrames(br, stdin, signalGroup, kill)
	} else {
		go func() {
			_, cerr := io.Copy(stdin, br)
			_ = stdin.Close()
			if cerr != nil {
				kill()
			}
		}()
	}

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
					kill()
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

// readClientFrames consumes the V2 client->server stream: stdin chunks, an
// explicit stdin-EOF, and signal frames forwarded to the process group. A read
// error means the client vanished, so the group is reaped.
func readClientFrames(br *bufio.Reader, stdin io.WriteCloser, signal func(syscall.Signal), kill func()) {
	eofed := false
	for {
		t, p, err := proto.ReadFrame(br)
		if err != nil {
			if !eofed {
				kill() // client disconnected before finishing input
			}
			return
		}
		switch t {
		case proto.FrameStdin:
			_, _ = stdin.Write(p)
		case proto.FrameStdinEOF:
			_ = stdin.Close()
			eofed = true
		case proto.FrameSignal:
			if len(p) > 0 {
				signal(syscall.Signal(int(p[0])))
			}
		}
	}
}

// handleRunDir serves `knit run --dir[/--sync]`: it accepts the request, receives
// the client's working directory as a streamed tar into a temp dir, runs the
// command there, and (for --sync) mirrors changed files back by content hash
// before the exit frame. Because the ok envelope is sent before the tree
// arrives, spawn errors here are reported as a stderr frame plus a non-zero exit.
func handleRunDir(conn net.Conn, br *bufio.Reader, req proto.Request) {
	tmp, err := os.MkdirTemp("", "knit-run-*")
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeInternal, Error: err.Error()})
		return
	}
	defer os.RemoveAll(tmp)

	writeEnvelope(conn, proto.Envelope{OK: true}) // accept; the tree streams next
	fw := proto.NewFrameWriter(conn)

	if err := recvTree(br, tmp); err != nil {
		_ = fw.Write(proto.FrameStderr, []byte("knit: receiving --dir tree: "+err.Error()+"\n"))
		_ = fw.WriteExit(1)
		return
	}
	var snap map[string]string
	if req.Sync {
		snap, _ = treesync.Snapshot(tmp)
	}

	c := exec.Command(req.Cmd[0], req.Cmd[1:]...)
	c.Dir = tmp
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, _ := c.StdinPipe()
	stdout, _ := c.StdoutPipe()
	stderr, _ := c.StderrPipe()
	if err := c.Start(); err != nil {
		_ = fw.Write(proto.FrameStderr, []byte("knit: "+err.Error()+"\n"))
		_ = fw.WriteExit(127)
		return
	}
	log.Printf("run --dir: %v", req.Cmd)

	pid := c.Process.Pid
	var killOnce sync.Once
	kill := func() {
		killOnce.Do(func() {
			_ = syscall.Kill(-pid, syscall.SIGINT)
			time.AfterFunc(2*time.Second, func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
		})
	}
	go readClientFrames(br, stdin, func(sig syscall.Signal) { _ = syscall.Kill(-pid, sig) }, kill)

	done := make(chan struct{}, 2)
	pump := func(t byte, r io.Reader) {
		bufp := pumpPool.Get().(*[]byte)
		buf := *bufp
		defer pumpPool.Put(bufp)
		for {
			n, rerr := r.Read(buf)
			if n > 0 {
				if fw.Write(t, buf[:n]) != nil {
					kill()
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
	if req.Sync {
		if changed, err := treesync.ChangedSince(tmp, snap); err == nil && len(changed) > 0 {
			sendTree(fw, tmp, changed)
		}
	}
	_ = fw.WriteExit(code)
}

// recvTree reads FrameInTar chunks until FrameInEnd, unpacking the stream into
// dst as it arrives (never buffering the whole tree).
func recvTree(br *bufio.Reader, dst string) error {
	pr, pw := io.Pipe()
	errc := make(chan error, 1)
	go func() { errc <- treesync.ReadTar(pr, dst) }()
	for {
		t, p, err := proto.ReadFrame(br)
		if err != nil {
			pw.CloseWithError(err)
			<-errc
			return err
		}
		switch t {
		case proto.FrameInTar:
			if _, werr := pw.Write(p); werr != nil {
				<-errc
				return werr
			}
		case proto.FrameInEnd:
			pw.Close()
			return <-errc
		}
	}
}

// sendTree streams the named changed files back to the client as FrameOutTar
// chunks followed by FrameOutEnd.
func sendTree(fw *proto.FrameWriter, root string, rels []string) {
	pr, pw := io.Pipe()
	go func() { pw.CloseWithError(treesync.WriteTarFiles(pw, root, rels)) }()
	buf := make([]byte, 32*1024)
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			if fw.Write(proto.FrameOutTar, buf[:n]) != nil {
				return
			}
		}
		if err != nil {
			break
		}
	}
	_ = fw.Write(proto.FrameOutEnd, nil)
}
