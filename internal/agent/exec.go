package agent

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/oddurs/knit/internal/paths"
	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/treesync"
)

const pumpBufSize = 64 * 1024

var pumpPool = sync.Pool{New: func() any { b := make([]byte, pumpBufSize); return &b }}

// handleRun spawns the requested command and streams its stdio over KNIT2
// frames: the client sends stdin chunks, an explicit stdin-EOF, and signal
// frames; the agent sends stdout, stderr, and a final exit frame. The command
// runs in its own process group so the whole tree can be reaped if the client
// goes away for any reason.
func handleRun(conn net.Conn, br *bufio.Reader, req proto.Request, proof string) {
	if len(req.Cmd) == 0 {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeEmptyCmd, Error: "empty command"})
		return
	}
	if req.V < proto.Version {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeVersion,
			Error: fmt.Sprintf("client speaks protocol v%d, this agent needs v%d — upgrade knit on the client", req.V, proto.Version)})
		return
	}
	if req.Dir {
		handleRunDir(conn, br, req, proof)
		return
	}
	c := command(req.Cmd)
	if home, err := os.UserHomeDir(); err == nil {
		c.Dir = home
	}
	cleanup, err := rankEnv(c, req)
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeInternal, Error: err.Error()})
		return
	}
	defer cleanup()
	pipes, err := start(c)
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeSpawn, Error: err.Error()})
		return
	}
	writeEnvelope(conn, proto.Envelope{OK: true, Proof: proof})
	log.Printf("run: %v", req.Cmd)

	fw := proto.NewFrameWriter(conn)
	_ = fw.WriteExit(stream(fw, br, c, pipes))
}

// handleRunDir serves `knit run --dir[/--sync]`: it accepts the request, receives
// the client's working directory as a streamed tar into a temp dir, runs the
// command there, and (for --sync) mirrors changed files back by content hash
// before the exit frame. Because the ok envelope is sent before the tree
// arrives, spawn errors here are reported as a stderr frame plus a non-zero exit.
func handleRunDir(conn net.Conn, br *bufio.Reader, req proto.Request, proof string) {
	tmp, err := os.MkdirTemp("", "knit-run-*")
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeInternal, Error: err.Error()})
		return
	}
	defer os.RemoveAll(tmp)

	writeEnvelope(conn, proto.Envelope{OK: true, Proof: proof}) // accept; the tree streams next
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

	c := command(req.Cmd)
	c.Dir = tmp
	cleanup, err := rankEnv(c, req)
	if err != nil {
		_ = fw.Write(proto.FrameStderr, []byte("knit: "+err.Error()+"\n"))
		_ = fw.WriteExit(1)
		return
	}
	defer cleanup()
	pipes, err := start(c)
	if err != nil {
		_ = fw.Write(proto.FrameStderr, []byte("knit: "+err.Error()+"\n"))
		_ = fw.WriteExit(127)
		return
	}
	log.Printf("run --dir: %v", req.Cmd)

	code := stream(fw, br, c, pipes)
	if req.Sync {
		if changed, err := treesync.ChangedSince(tmp, snap); err == nil && len(changed) > 0 {
			sendTree(fw, tmp, changed)
		}
	}
	_ = fw.WriteExit(code)
}

// stdio is a started command's three pipes.
type stdio struct {
	in       io.WriteCloser
	out, err io.Reader
}

// command builds the exec.Cmd for cmd in its own process group.
func command(cmd []string) *exec.Cmd {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return c
}

// rankEnv gives c the rank environment of a `knit each` launch when the
// request carries one; the returned cleanup removes the hostfile it wrote.
func rankEnv(c *exec.Cmd, req proto.Request) (func(), error) {
	if len(req.Hosts) == 0 {
		return func() {}, nil
	}
	env, cleanup, err := paths.RankEnv(req.Rank, req.Hosts)
	if err != nil {
		return nil, err
	}
	c.Env = append(os.Environ(), env...)
	return cleanup, nil
}

// start wires the pipes and starts c.
func start(c *exec.Cmd) (stdio, error) {
	var s stdio
	var err error
	if s.in, err = c.StdinPipe(); err != nil {
		return s, err
	}
	if s.out, err = c.StdoutPipe(); err != nil {
		return s, err
	}
	if s.err, err = c.StderrPipe(); err != nil {
		return s, err
	}
	return s, c.Start()
}

// stream pumps a started command's stdout and stderr to the client as frames
// while consuming the client's framed stdin and signals, until the command
// exits. It returns the exit code. Losing the client at any point — before or
// after stdin-EOF, with or without output pending — reaps the process group, so
// a dropped link never leaves an orphan.
func stream(fw *proto.FrameWriter, br *bufio.Reader, c *exec.Cmd, s stdio) int {
	pid := c.Process.Pid
	finished := make(chan struct{})
	var killOnce sync.Once
	kill := func() {
		killOnce.Do(func() {
			select {
			case <-finished:
				return // exited on its own; the pid may already be reused
			default:
			}
			_ = syscall.Kill(-pid, syscall.SIGINT)
			go func() {
				select {
				case <-finished:
				case <-time.After(2 * time.Second):
					_ = syscall.Kill(-pid, syscall.SIGKILL)
				}
			}()
		})
	}
	signalGroup := func(sig syscall.Signal) { _ = syscall.Kill(-pid, sig) }
	go readClientFrames(br, s.in, signalGroup, kill)

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
	go pump(proto.FrameStdout, s.out)
	go pump(proto.FrameStderr, s.err)
	<-done
	<-done

	code := proto.ExitStatus(c.Wait())
	close(finished)
	return code
}

// readClientFrames consumes the client->server stream: stdin chunks, an
// explicit stdin-EOF, and signal frames forwarded to the process group. The
// stream ends only when the client goes away, so a read error always reaps.
func readClientFrames(br *bufio.Reader, stdin io.WriteCloser, signal func(syscall.Signal), kill func()) {
	for {
		t, p, err := proto.ReadFrame(br)
		if err != nil {
			kill()
			return
		}
		switch t {
		case proto.FrameStdin:
			_, _ = stdin.Write(p)
		case proto.FrameStdinEOF:
			_ = stdin.Close()
		case proto.FrameSignal:
			if len(p) > 0 {
				signal(syscall.Signal(int(p[0])))
			}
		}
	}
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
