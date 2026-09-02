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

// Run schedules cmd and streams it. With onName set it pins that machine;
// otherwise it scores the local machine against reachable peers and picks the
// least-loaded. A local win execs in-process and prints nothing; a remote run
// prints one dim line to stderr. The return value is the command's exit code,
// or a knit exit code on a knit-level failure.
func Run(onName string, dirMode, sync bool, cmd []string) int {
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "knit: run needs a command after --")
		return ExitUsage
	}
	if sync {
		dirMode = true // can't mirror back what was never sent
	}
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}

	peers := probePeers(key, false)

	if onName != "" {
		if c, ok := scheduler.ByName(peers, onName); ok {
			return runRemote(c, key, cmd, dirMode, sync)
		}
		if onName == localCandidate().Name {
			return runLocal(cmd)
		}
		fmt.Fprintf(os.Stderr, "knit: no reachable machine named %q\n", onName)
		return ExitUnreachable
	}

	cands := append([]scheduler.Candidate{localCandidate()}, peers...)
	best, ok := scheduler.Pick(cands)
	if !ok || best.Local {
		return runLocal(cmd) // already runs in the local working directory
	}
	return runRemote(best, key, cmd, dirMode, sync)
}

// runLocal executes the command in this process's environment, inheriting
// stdio, and returns its exit code. knit prints nothing — the invisible fallback.
func runLocal(cmd []string) int {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}
	return 0
}

// runRemote dials the target and streams stdio over the KNIT2 framed protocol:
// stdin as frames with an explicit EOF, Ctrl-C/SIGTERM forwarded as signal
// frames, and stdout/stderr/exit coming back. The command's exit code becomes
// this process's exit code.
func runRemote(c scheduler.Candidate, key []byte, cmd []string, dirMode, sync bool) int {
	req := proto.Request{Op: proto.OpRun, Cmd: cmd, Dir: dirMode, Sync: sync}
	sess, err := transport.Open(c.HostPortOrEmpty(), key, req, dialTimeout)
	if err != nil {
		if re, ok := err.(*transport.ReplyError); ok && re.Code == proto.CodeUnauthorized {
			fmt.Fprintf(os.Stderr, "knit: unauthorized on %s — run `knit key` there and `knit join <key>` here\n", c.Name)
			return ExitUnauthorized
		}
		fmt.Fprintf(os.Stderr, "knit: cannot reach %s: %v\n", c.Name, err)
		return ExitUnreachable
	}
	defer sess.Close()

	fmt.Fprintf(os.Stderr, "%s\n", dim("knit → "+c.Name))

	fw := proto.NewFrameWriter(sess.Conn)

	// --dir: stream the working directory to the target before anything else.
	if dirMode {
		if cwd, err := os.Getwd(); err == nil {
			pr, pw := io.Pipe()
			go func() { pw.CloseWithError(treesync.WriteTar(pw, cwd)) }()
			buf := make([]byte, 32*1024)
			for {
				n, rerr := pr.Read(buf)
				if n > 0 {
					_ = fw.Write(proto.FrameInTar, buf[:n])
				}
				if rerr != nil {
					break
				}
			}
			_ = fw.Write(proto.FrameInEnd, nil)
		}
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

func dim(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}
