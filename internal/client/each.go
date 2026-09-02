package client

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"

	"github.com/oddurs/knit/internal/discovery"
	"github.com/oddurs/knit/internal/paths"
	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/scheduler"
	"github.com/oddurs/knit/internal/sysinfo"
	"github.com/oddurs/knit/internal/transport"
)

// Each runs cmd on every machine at once — this one in-process, every
// reachable, authorized peer over the wire — prefixing each output line with
// the machine name. It exits 0 only if every machine exited 0, else the
// highest code observed.
//
// Every process is launched with the rank environment (proto.RankEnv): this
// machine is rank 0 and peers follow in order of spare capacity, so torchrun
// and MLX distributed start from `knit each` with no hostfile written by hand.
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
	peers, _ := probePeers(key, true)
	sort.SliceStable(peers, func(i, j int) bool { return peers[i].Info.Score() < peers[j].Info.Score() })
	hosts := make([]string, 0, 1+len(peers))
	if len(peers) > 0 {
		hosts = append(hosts, discovery.LocalAddrToward(peers[0].Addr))
	} else {
		hosts = append(hosts, discovery.LocalAddrToward(""))
	}
	for _, p := range peers {
		hosts = append(hosts, p.Addr)
	}

	var mu sync.Mutex // serialize writes to the shared stdout/stderr
	var wg sync.WaitGroup
	names := []string{sysinfo.Name()}
	codes := make([]int, 1+len(peers))
	wg.Add(1)
	go func() {
		defer wg.Done()
		codes[0] = eachLocal(names[0], cmd, hosts, &mu)
	}()
	for i, p := range peers {
		names = append(names, p.Name)
		wg.Add(1)
		go func(i int, c scheduler.Candidate) {
			defer wg.Done()
			codes[i] = eachOne(c, key, proto.Request{Op: proto.OpRun, Cmd: cmd, Hosts: hosts, Rank: i}, &mu)
		}(i+1, p)
	}
	wg.Wait()

	worst := 0
	summary := make([]string, 0, len(names))
	for i, name := range names {
		if codes[i] > worst {
			worst = codes[i]
		}
		summary = append(summary, fmt.Sprintf("%s=%d", name, codes[i]))
	}
	fmt.Fprintf(os.Stderr, "%s\n", dim("knit each: "+strings.Join(summary, " ")))
	return worst
}

// eachLocal runs cmd on this machine (rank 0) with the same prefixing and no
// stdin, so the local line looks exactly like a peer's.
func eachLocal(name string, cmd []string, hosts []string, mu *sync.Mutex) int {
	out := &prefixer{mu: mu, w: os.Stdout, prefix: "[" + name + "] "}
	errs := &prefixer{mu: mu, w: os.Stderr, prefix: "[" + name + "] "}
	defer out.flush()
	defer errs.flush()
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout, c.Stderr = out, errs
	if env, cleanup, err := paths.RankEnv(0, hosts); err == nil {
		defer cleanup()
		c.Env = append(os.Environ(), env...)
	}
	err := c.Run()
	if _, exited := err.(*exec.ExitError); err != nil && !exited {
		errs.Write([]byte("knit: " + err.Error() + "\n"))
	}
	return proto.ExitStatus(err)
}

func eachOne(c scheduler.Candidate, key []byte, req proto.Request, mu *sync.Mutex) int {
	sess, err := transport.Open(c.HostPortOrEmpty(), key, req, dialTimeout)
	if err != nil {
		mu.Lock()
		defer mu.Unlock()
		return reportOpenError(c.Name, err)
	}
	defer sess.Close()
	// each sends no stdin; an explicit EOF frame tells the agent so.
	_ = proto.NewFrameWriter(sess.Conn).Write(proto.FrameStdinEOF, nil)

	prefix := "[" + c.Name + "] "
	out := &prefixer{mu: mu, w: os.Stdout, prefix: prefix}
	errs := &prefixer{mu: mu, w: os.Stderr, prefix: prefix}
	defer out.flush()
	defer errs.flush()
	for {
		t, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			return ExitDisconnected
		}
		switch t {
		case proto.FrameStdout:
			out.Write(p)
		case proto.FrameStderr:
			errs.Write(p)
		case proto.FrameExit:
			return proto.ExitCode(p)
		}
	}
}

// prefixer writes a stream to w one whole line at a time, each prefixed. A
// line that straddles two frames is held until its newline arrives, so output
// from many machines interleaves only at line boundaries.
type prefixer struct {
	mu      *sync.Mutex
	w       io.Writer
	prefix  string
	partial []byte
}

// Write implements io.Writer; it never fails, since dropping a peer's output
// on a local stdout error would be worse than continuing.
func (p *prefixer) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(b)
	for {
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			p.partial = append(p.partial, b...)
			return n, nil
		}
		fmt.Fprintf(p.w, "%s%s%s", p.prefix, p.partial, b[:i+1])
		p.partial = p.partial[:0]
		b = b[i+1:]
	}
}

// flush emits a trailing line that never got its newline.
func (p *prefixer) flush() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.partial) > 0 {
		fmt.Fprintf(p.w, "%s%s\n", p.prefix, p.partial)
		p.partial = p.partial[:0]
	}
}
