// Package client is the client half of connex: discover agents, probe their
// live capacity, schedule a command, and stream its stdio. See docs/04-cli.md.
package client

import (
	"os"
	"strconv"
	"time"

	"github.com/oddurs/connex/internal/discovery"
	"github.com/oddurs/connex/internal/keys"
	"github.com/oddurs/connex/internal/proto"
	"github.com/oddurs/connex/internal/scheduler"
	"github.com/oddurs/connex/internal/sysinfo"
	"github.com/oddurs/connex/internal/transport"
)

// connex's own exit codes, disjoint from a command's 0-125 range so scripts can
// tell "the command failed" from "connex failed". See docs/04-cli.md.
const (
	ExitUsage        = 2
	ExitDisconnected = 124
	ExitUnreachable  = 126
	ExitUnauthorized = 127
)

const (
	browseTimeout  = 1 * time.Second
	defaultProbeMS = 250
	dialTimeout    = 3 * time.Second
)

func probeTimeout() time.Duration {
	if v := os.Getenv("CONNEX_TIMEOUT_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultProbeMS * time.Millisecond
}

// localCandidate builds the always-present local scheduling candidate.
func localCandidate() scheduler.Candidate {
	info := sysinfo.Local()
	return scheduler.Candidate{Name: info.Name, Local: true, Info: info}
}

// probePeers browses (cached) for agents, drops any that share this machine's
// name (our own advertised agent), probes the rest for live capacity in
// parallel within the probe budget, and returns the reachable, authorized ones.
func probePeers(key []byte, fresh bool) []scheduler.Candidate {
	self := sysinfo.Local().Name
	var peers []discovery.Peer
	if fresh {
		peers = discovery.Fresh(browseTimeout)
	} else {
		peers = discovery.CachedBrowse(browseTimeout)
	}

	type result struct{ c scheduler.Candidate }
	ch := make(chan *scheduler.Candidate, len(peers))
	budget := probeTimeout()
	n := 0
	for _, p := range peers {
		if p.Name == self {
			continue // our own agent
		}
		n++
		go func(p discovery.Peer) {
			info, err := probeInfo(p, key, budget)
			if err != nil {
				ch <- nil
				return
			}
			ch <- &scheduler.Candidate{Name: info.Name, Addr: p.Addr, Port: p.Port, Info: info}
		}(p)
	}
	var out []scheduler.Candidate
	for i := 0; i < n; i++ {
		if c := <-ch; c != nil {
			out = append(out, *c)
		}
	}
	return out
}

func probeInfo(p discovery.Peer, key []byte, timeout time.Duration) (proto.Envelope, error) {
	sess, err := transport.Open(p.HostPort(), key, proto.Request{Op: proto.OpInfo}, timeout)
	if err != nil {
		return proto.Envelope{}, err
	}
	defer sess.Close()
	return sess.Reply, nil
}

func loadKey() ([]byte, error) { return keys.LoadOrCreate() }
