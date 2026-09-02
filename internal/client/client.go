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

// gatherPeers merges mDNS-discovered peers (minus our own advertised agent) with
// explicit peers from --peer and KNIT_PEERS, de-duplicated by host:port. Explicit
// peers make knit work where multicast is unavailable, e.g. a Tailscale tailnet.
func gatherPeers(fresh bool) []discovery.Peer {
	self := sysinfo.Local().Name
	var out []discovery.Peer
	seen := map[string]bool{}
	add := func(p discovery.Peer) {
		hp := p.HostPort()
		if seen[hp] {
			return
		}
		seen[hp] = true
		out = append(out, p)
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
		add(p)
	}
	for _, p := range discovery.EnvPeers() {
		add(p)
	}
	for _, p := range discovery.ParsePeerList(extraPeers) {
		add(p)
	}
	return out
}

// probePeers gathers candidate peers and probes each for live capacity in
// parallel within the probe budget, returning the reachable, authorized ones
// with a best-effort link classification set from the peer's address.
func probePeers(key []byte, fresh bool) []scheduler.Candidate {
	peers := gatherPeers(fresh)
	ch := make(chan *scheduler.Candidate, len(peers))
	budget := probeTimeout()
	for _, p := range peers {
		go func(p discovery.Peer) {
			info, err := probeInfo(p, key, budget)
			if err != nil {
				ch <- nil
				return
			}
			info.Link = discovery.ClassifyLink(p.Addr)
			ch <- &scheduler.Candidate{Name: info.Name, Addr: p.Addr, Port: p.Port, Info: info}
		}(p)
	}
	var res []scheduler.Candidate
	for range peers {
		if c := <-ch; c != nil {
			res = append(res, *c)
		}
	}
	return res
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
