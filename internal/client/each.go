package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/scheduler"
	"github.com/oddurs/knit/internal/transport"
)

// Each runs cmd on every reachable, authorized agent concurrently, prefixing
// each output line with the machine name. It exits 0 only if every machine
// exited 0, else the highest code observed.
func Each(cmd []string) int {
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "knit: each needs a command after --")
		return ExitUsage
	}
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}
	peers := probePeers(key, true)
	if len(peers) == 0 {
		fmt.Fprintln(os.Stderr, "knit: no other machines found")
		return ExitUnreachable
	}

	var mu sync.Mutex // serialize writes to the shared stdout/stderr
	var wg sync.WaitGroup
	codes := make([]int, len(peers))
	for i, p := range peers {
		wg.Add(1)
		go func(i int, c scheduler.Candidate) {
			defer wg.Done()
			codes[i] = eachOne(c, key, cmd, &mu)
		}(i, p)
	}
	wg.Wait()

	worst := 0
	summary := make([]string, 0, len(peers))
	for i, c := range peers {
		if codes[i] > worst {
			worst = codes[i]
		}
		summary = append(summary, fmt.Sprintf("%s=%d", c.Name, codes[i]))
	}
	fmt.Fprintf(os.Stderr, "%s\n", dim("knit each: "+strings.Join(summary, " ")))
	return worst
}

func eachOne(c scheduler.Candidate, key []byte, cmd []string, mu *sync.Mutex) int {
	sess, err := transport.Open(c.HostPortOrEmpty(), key, proto.Request{Op: proto.OpRun, Cmd: cmd}, dialTimeout)
	if err != nil {
		mu.Lock()
		fmt.Fprintf(os.Stderr, "[%s] knit: %v\n", c.Name, err)
		mu.Unlock()
		return ExitUnreachable
	}
	defer sess.Close()
	// each sends no stdin; an explicit EOF frame tells the agent so.
	_ = proto.NewFrameWriter(sess.Conn).Write(proto.FrameStdinEOF, nil)

	prefix := "[" + c.Name + "] "
	for {
		t, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			return ExitDisconnected
		}
		switch t {
		case proto.FrameStdout:
			writePrefixed(mu, os.Stdout, prefix, p)
		case proto.FrameStderr:
			writePrefixed(mu, os.Stderr, prefix, p)
		case proto.FrameExit:
			return proto.ExitCode(p)
		}
	}
}

// writePrefixed writes each line of p to w with the given prefix, under mu.
func writePrefixed(mu *sync.Mutex, w io.Writer, prefix string, p []byte) {
	mu.Lock()
	defer mu.Unlock()
	sc := bufio.NewScanner(bytes.NewReader(p))
	sc.Buffer(make([]byte, 0, 64*1024), proto.MaxFrame)
	for sc.Scan() {
		fmt.Fprintf(w, "%s%s\n", prefix, sc.Text())
	}
}
