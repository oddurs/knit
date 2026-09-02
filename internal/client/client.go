// Package client is the client half of knit: discover agents, probe their
// live capacity, schedule a command, and stream its stdio. See docs/04-cli.md.
package client

import (
	"os"
	"strconv"
	"time"

	"github.com/oddurs/knit/internal/discovery"
	"github.com/oddurs/knit/internal/keys"
	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/scheduler"
	"github.com/oddurs/knit/internal/sysinfo"
	"github.com/oddurs/knit/internal/transport"
)

// knit's own exit codes, disjoint from a command's 0-125 range so scripts can
// tell "the command failed" from "knit failed". See docs/04-cli.md.
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
	if v := os.Getenv("KNIT_TIMEOUT_MS"); v != "" {
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

// extraPeers holds explicit peers added via the --peer flag (see AddExplicitPeers).
var extraPeers []string

// AddExplicitPeers registers peers given on the command line with --peer. They
// are probed alongside (and merged with) mDNS-discovered peers.
func AddExplicitPeers(hostPorts []string) { extraPeers = append(extraPeers, hostPorts...) }

// peer is a dial target and whether the user named it (--peer, KNIT_PEERS)
// rather than discovery finding it.
type peer struct {
	discovery.Peer
	explicit bool
}

// gatherPeers merges mDNS-discovered peers (minus our own advertised agent) with
// explicit peers from --peer and KNIT_PEERS, de-duplicated by host:port. Explicit
// peers make knit work where multicast is unavailable, e.g. a Tailscale tailnet.
func gatherPeers(fresh bool) []peer {
	self := sysinfo.Name()
	var out []peer
	seen := map[string]bool{}
	add := func(p discovery.Peer, explicit bool) {
		hp := p.HostPort()
		if seen[hp] {
			return
		}
		seen[hp] = true
		out = append(out, peer{p, explicit})
	}

	var mdns []discovery.Peer
	if fresh {
		mdns = discovery.Fresh(browseTimeout)
	} else {
		mdns = discovery.CachedBrowse(browseTimeout)
	}
	for _, p := range mdns {
		if p.Name == self {
			continue // our own agent
		}
		add(p, false)
	}
	for _, p := range discovery.EnvPeers() {
		add(p, true)
	}
	for _, p := range discovery.ParsePeerList(extraPeers) {
		add(p, true)
	}
	return out
}

// probeFailure is an explicitly named peer that could not be used, and why.
// Discovered peers that fail are dropped silently (a stale cache entry is
// normal); a peer the user typed deserves an explanation.
type probeFailure struct {
	HostPort string
	Err      error
}

// probePeers gathers candidate peers and probes each for live capacity in
// parallel within the probe budget, returning the reachable, authorized ones
// with the link they are reached over classified from the peer's address,
// plus the failures among explicitly named peers.
func probePeers(key []byte, fresh bool) ([]scheduler.Candidate, []probeFailure) {
	peers := gatherPeers(fresh)
	type result struct {
		cand *scheduler.Candidate
		fail *probeFailure
	}
	ch := make(chan result, len(peers))
	budget := probeTimeout()
	for _, p := range peers {
		go func(p peer) {
			info, err := probeInfo(p.Peer, key, budget)
			if err != nil {
				var f *probeFailure
				if p.explicit {
					f = &probeFailure{p.HostPort(), err}
				}
				ch <- result{fail: f}
				return
			}
			label, mbps := discovery.Link(p.Addr)
			info.Link = label
			ch <- result{cand: &scheduler.Candidate{Name: info.Name, Addr: p.Addr, Port: p.Port, Info: info, LinkMbps: mbps}}
		}(p)
	}
	var res []scheduler.Candidate
	var fails []probeFailure
	for range peers {
		r := <-ch
		if r.cand != nil {
			res = append(res, *r.cand)
		}
		if r.fail != nil {
			fails = append(fails, *r.fail)
		}
	}
	return res, fails
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
