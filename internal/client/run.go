package client

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/scheduler"
	"github.com/oddurs/knit/internal/transport"
	"github.com/oddurs/knit/internal/treesync"
)

// Placement holds the run flags that decide where a command goes: a pinned
// machine (--on), a minimum of free memory in GB (--mem), an architecture
// (--arch), and whether the working directory travels with it (--dir/--sync).
type Placement struct {
	On    string
	MemGB float64
	Arch  string
	Dir   bool
	Sync  bool
}

// Run schedules cmd and streams it. With On set it pins that machine;
// otherwise it scores the local machine against reachable peers and picks the
// least-loaded that satisfies the constraints. A local win execs in-process
// and prints nothing; a remote run prints one dim line to stderr. The return
// value is the command's exit code, or a knit exit code on a knit-level
// failure.
func Run(p Placement, cmd []string) int {
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "knit: run needs a command after --")
		return ExitUsage
	}
	if p.Sync {
		p.Dir = true // can't mirror back what was never sent
	}
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}

	cands := append([]scheduler.Candidate{localCandidate()}, probePeers(key, false)...)
	cands, why := scheduler.Filter(cands, p.MemGB, p.Arch)
	if why != "" {
		fmt.Fprintln(os.Stderr, "knit:", why)
		return ExitUnreachable
	}

	var best scheduler.Candidate
	if p.On != "" {
		c, ok := scheduler.ByName(cands, p.On)
		if !ok {
			fmt.Fprintf(os.Stderr, "knit: no reachable machine named %q%s\n", p.On, constraintNote(p))
			return ExitUnreachable
		}
		best = c
	} else {
		best, _ = scheduler.Pick(cands) // never empty: Filter reported otherwise
	}
	if best.Local {
		return runLocal(cmd) // already runs in the local working directory
	}
	return runRemote(best, key, cmd, p.Dir, p.Sync)
}

// constraintNote explains, when --on misses, that a constraint may be why.
func constraintNote(p Placement) string {
	switch {
	case p.MemGB > 0 && p.Arch != "":
		return fmt.Sprintf(" with %g GB free and arch %s", p.MemGB, p.Arch)
	case p.MemGB > 0:
		return fmt.Sprintf(" with %g GB free", p.MemGB)
	case p.Arch != "":
		return " with arch " + p.Arch
	}
	return ""
}

// runLocal executes the command in this process's environment, inheriting
// stdio, and returns its exit code. knit prints nothing — the invisible fallback.
func runLocal(cmd []string) int {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := c.Run()
	if _, exited := err.(*exec.ExitError); err != nil && !exited {
		fmt.Fprintln(os.Stderr, "knit:", err)
	}
	return proto.ExitStatus(err)
}

// runRemote dials the target and streams stdio over the KNIT2 framed protocol:
// stdin as frames with an explicit EOF, Ctrl-C/SIGTERM forwarded as signal
// frames, and stdout/stderr/exit coming back. The command's exit code becomes
// this process's exit code.
func runRemote(c scheduler.Candidate, key []byte, cmd []string, dirMode, sync bool) int {
	req := proto.Request{Op: proto.OpRun, Cmd: cmd, Dir: dirMode, Sync: sync}
	sess, err := transport.Open(c.HostPortOrEmpty(), key, req, dialTimeout)
	if err != nil {
		return reportOpenError(c.Name, err)
	}
	defer sess.Close()

	fmt.Fprintf(os.Stderr, "%s\n", dim("knit → "+c.Name))

	fw := proto.NewFrameWriter(sess.Conn)

	// --dir: stream the working directory to the target before anything else.
	// The agent waits for FrameInEnd, so it is always sent — even for an empty
	// or unreadable tree — and a dead link stops the walk instead of draining it.
	if dirMode {
		if cwd, err := os.Getwd(); err == nil {
			pr, pw := io.Pipe()
			go func() { pw.CloseWithError(treesync.WriteTar(pw, cwd)) }()
			buf := make([]byte, 32*1024)
			for {
				n, rerr := pr.Read(buf)
				if n > 0 {
					if fw.Write(proto.FrameInTar, buf[:n]) != nil {
						pr.Close()
						break
					}
				}
				if rerr != nil {
					break
				}
			}
		}
		_ = fw.Write(proto.FrameInEnd, nil)
	}

	// Forward stdin as frames, then an explicit EOF frame.
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, rerr := os.Stdin.Read(buf)
			if n > 0 {
				if fw.Write(proto.FrameStdin, buf[:n]) != nil {
					return
				}
			}
			if rerr != nil {
				break
			}
		}
		_ = fw.Write(proto.FrameStdinEOF, nil)
	}()

	// Forward SIGINT/SIGTERM to the remote process as signal frames.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for s := range sig {
			n := byte(int(syscall.SIGINT))
			if s == syscall.SIGTERM {
				n = byte(int(syscall.SIGTERM))
			}
			_ = fw.Write(proto.FrameSignal, []byte{n})
		}
	}()

	var syncPw *io.PipeWriter
	var syncErr chan error
	for {
		t, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			fmt.Fprintln(os.Stderr, "knit: connection lost before command finished")
			return ExitDisconnected
		}
		switch t {
		case proto.FrameStdout:
			os.Stdout.Write(p)
		case proto.FrameStderr:
			os.Stderr.Write(p)
		case proto.FrameOutTar:
			if syncPw == nil {
				cwd, _ := os.Getwd()
				pr, pw := io.Pipe()
				syncPw = pw
				syncErr = make(chan error, 1)
				go func() { syncErr <- treesync.ReadTar(pr, cwd) }()
			}
			_, _ = syncPw.Write(p)
		case proto.FrameOutEnd:
			if syncPw != nil {
				syncPw.Close()
				if e := <-syncErr; e != nil {
					fmt.Fprintln(os.Stderr, "knit: applying --sync changes:", e)
				}
				syncPw = nil
			}
		case proto.FrameExit:
			return proto.ExitCode(p)
		}
	}
}

// reportOpenError prints the reason a run could not start on name and returns
// the matching knit exit code: unauthorized (127), a command the target could
// not spawn (127, as a local shell reports a missing program), or unreachable.
func reportOpenError(name string, err error) int {
	if re, ok := err.(*transport.ReplyError); ok {
		switch re.Code {
		case proto.CodeUnauthorized:
			fmt.Fprintf(os.Stderr, "knit: unauthorized on %s — run `knit key` there and `knit join <key>` here\n", name)
			return ExitUnauthorized
		case proto.CodeSpawn:
			fmt.Fprintf(os.Stderr, "knit: on %s: %s\n", name, re.Msg)
			return 127
		}
	}
	fmt.Fprintf(os.Stderr, "knit: cannot reach %s: %v\n", name, err)
	return ExitUnreachable
}

func dim(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}
