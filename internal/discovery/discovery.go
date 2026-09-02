// Package discovery finds knit agents over multicast-DNS on every interface,
// and registers this machine's agent. A short client-side cache (KN-DISC-002)
// keeps back-to-back commands from each paying the browse latency; capacity is
// never cached, only addresses. See docs/adr/0003-mdns-discovery-and-cache.md.
package discovery

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/grandcat/zeroconf"
)

// ServiceType is the mDNS service knit agents advertise.
const ServiceType = "_knit._tcp"

// Peer is a discovered agent: a name and where to reach it. Live capacity is
// obtained separately via an info probe, never from mDNS.
type Peer struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

// HostPort is the dial target for the peer.
func (p Peer) HostPort() string {
	return net.JoinHostPort(p.Addr, strconv.Itoa(p.Port))
}

// Register advertises this machine's agent and returns a handle whose Shutdown
// withdraws the advertisement.
func Register(name string, port int, txt []string) (*zeroconf.Server, error) {
	return zeroconf.Register(name, ServiceType, "local.", port, txt, nil)
}

// Browse performs a fresh mDNS browse for the given duration and returns the
// unique peers found, preferring IPv4 addresses.
func Browse(timeout time.Duration) []Peer {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil
	}
	entries := make(chan *zeroconf.ServiceEntry, 16)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := resolver.Browse(ctx, ServiceType, "local.", entries); err != nil {
		return nil
	}

	seen := map[string]bool{}
	var peers []Peer
	for {
		select {
		case e, ok := <-entries:
			if !ok {
				return peers
			}
			if e == nil || seen[e.Instance] {
				continue
			}
			addr := pickAddr(e)
			if addr == "" {
				continue
			}
			seen[e.Instance] = true
			peers = append(peers, Peer{Name: e.Instance, Addr: addr, Port: e.Port})
		case <-ctx.Done():
			return peers
		}
	}
}

func pickAddr(e *zeroconf.ServiceEntry) string {
	if len(e.AddrIPv4) > 0 {
		return e.AddrIPv4[0].String()
	}
	if len(e.AddrIPv6) > 0 {
		return e.AddrIPv6[0].String()
	}
	return ""
}
