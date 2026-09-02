package client

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/scheduler"
	"github.com/oddurs/knit/internal/transport"
)

// Run schedules cmd and streams it. With onName set it pins that machine;
// otherwise it scores the local machine against reachable peers and picks the
// least-loaded. A local win execs in-process and prints nothing; a remote run
// prints one dim line to stderr. The return value is the command's exit code,
// or a knit exit code on a knit-level failure.
func Run(onName string, cmd []string) int {
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "knit: run needs a command after --")
		return ExitUsage
	}
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}

	peers := probePeers(key, false)

	if onName != "" {
		if c, ok := scheduler.ByName(peers, onName); ok {
			return runRemote(c, key, cmd)
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
		return runLocal(cmd)
	}
	return runRemote(best, key, cmd)
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
func runRemote(c scheduler.Candidate, key []byte, cmd []string) int {
	sess, err := transport.Open(c.HostPortOrEmpty(), key, proto.Request{Op: proto.OpRun, Cmd: cmd}, dialTimeout)
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
