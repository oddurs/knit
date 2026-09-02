package discovery

import (
	"net"
	"os"
	"strconv"
	"strings"
)

// DefaultPort is the port an agent listens on when it is free (it spells KNIT
// on a phone keypad and is unassigned by IANA). An agent that finds it taken
// falls back to an ephemeral port, which mDNS still advertises; only explicit
// --peer targets need a stable port, and they get one.
const DefaultPort = 5648

// ParsePeer parses a "host" or "host:port" into a Peer with an empty Name (the
// name is learned from the peer's info reply). It works across a Tailscale
// tailnet or any network where multicast mDNS is unavailable. Without a port
// the peer is assumed to be on DefaultPort.
func ParsePeer(hostPort string) (Peer, bool) {
	hostPort = strings.TrimSpace(hostPort)
	if !strings.Contains(hostPort, ":") {
		if hostPort == "" {
			return Peer{}, false
		}
		return Peer{Addr: hostPort, Port: DefaultPort}, true
	}
	host, portStr, err := net.SplitHostPort(hostPort)
	if err != nil || host == "" {
		return Peer{}, false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return Peer{}, false
	}
	return Peer{Addr: host, Port: port}, true
}

// EnvPeers returns the peers listed in the KNIT_PEERS environment variable
// (comma-separated host[:port]), skipping any that do not parse.
func EnvPeers() []Peer {
	return ParsePeerList(strings.Split(os.Getenv("KNIT_PEERS"), ","))
}

// ParsePeerList parses a slice of "host[:port]" strings, dropping blanks and
// anything that does not parse.
func ParsePeerList(items []string) []Peer {
	var out []Peer
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		if p, ok := ParsePeer(it); ok {
			out = append(out, p)
		}
	}
	return out
}

// ClassifyLink guesses how a peer is reached from its address. This is a
// heuristic for gauge output: an IPv4/IPv6 link-local address is almost always a
// Thunderbolt/USB4 bridge, a private address is a LAN (Wi-Fi or Ethernet, which
// the address alone cannot distinguish — true interface/link speed is v0.3,
// KN-CLIENT-030). Loopback is the local agent.
func ClassifyLink(addr string) string {
	ip := net.ParseIP(addr)
	if ip == nil {
		return ""
	}
	switch {
	case ip.IsLoopback():
		return "local"
	case ip.IsLinkLocalUnicast(): // 169.254.0.0/16, fe80::/10
		return "thunderbolt"
	case ip.IsPrivate(): // 10/8, 172.16/12, 192.168/16, fc00::/7
		return "lan"
	default:
		return "net"
	}
}
