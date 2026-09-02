package client

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/oddurs/connex/internal/proto"
	"github.com/oddurs/connex/internal/scheduler"
	"github.com/oddurs/connex/internal/transport"
)

// Run schedules cmd and streams it. With onName set it pins that machine;
// otherwise it scores the local machine against reachable peers and picks the
// least-loaded. A local win execs in-process and prints nothing; a remote run
// prints one dim line to stderr. The return value is the command's exit code,
// or a connex exit code on a connex-level failure.
func Run(onName string, cmd []string) int {
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "connex: run needs a command after --")
		return ExitUsage
	}
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "connex:", err)
		return 1
	}

	peers := probePeers(key, false)

	// Pin with --on.
	if onName != "" {
		if c, ok := scheduler.ByName(peers, onName); ok {
			return runRemote(c, key, cmd)
		}
		if onName == localCandidate().Name {
			return runLocal(cmd)
		}
		fmt.Fprintf(os.Stderr, "connex: no reachable machine named %q\n", onName)
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
// stdio, and returns its exit code. connex prints nothing — the invisible
// fallback.
func runLocal(cmd []string) int {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "connex:", err)
		return 1
	}
	return 0
}

// runRemote dials the target, streams stdio, and relays the exit code.
func runRemote(c scheduler.Candidate, key []byte, cmd []string) int {
	sess, err := transport.Open(c.HostPortOrEmpty(), key, proto.Request{Op: proto.OpRun, Cmd: cmd}, dialTimeout)
	if err != nil {
		if re, ok := err.(*transport.ReplyError); ok && re.Code == proto.CodeUnauthorized {
			fmt.Fprintf(os.Stderr, "connex: unauthorized on %s — run `connex key` there and `connex join <key>` here\n", c.Name)
			return ExitUnauthorized
		}
		fmt.Fprintf(os.Stderr, "connex: cannot reach %s: %v\n", c.Name, err)
		return ExitUnreachable
	}
	defer sess.Close()

	fmt.Fprintf(os.Stderr, "%s\n", dim("connex → "+c.Name))

	// Forward stdin; half-close on EOF so the remote process sees EOF.
	go func() {
		_, _ = io.Copy(sess.Conn, os.Stdin)
		if tc, ok := sess.Conn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	// Ctrl-C: closing the connection makes the agent reap the remote process.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		sess.Conn.Close()
	}()

	for {
		t, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			fmt.Fprintln(os.Stderr, "connex: connection lost before command finished")
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
