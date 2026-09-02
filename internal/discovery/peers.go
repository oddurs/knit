package discovery

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/oddurs/knit/internal/sysinfo"
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

// Link reports how a peer at addr is reached: a display label for gauge and a
// nominal speed in Mbps for the scheduler's tie-break. It looks up which local
// interface shares a subnet with the address and asks the OS what that
// interface is; an address reached through a router or overlay (Tailscale) is
// "net", a directly connected interface the OS cannot describe is "lan".
func Link(addr string) (label string, mbps int) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", 0
	}
	if ip.IsLoopback() {
		return "local", 0
	}
	iface, _ := interfaceToward(ip)
	if iface == "" {
		return "net", 0
	}
	l := sysinfo.LinkOf(iface)
	switch l.Kind {
	case "":
		return "lan", 0
	case "thunderbolt":
		return "thunderbolt ~" + speedLabel(l.Mbps), l.Mbps
	case "ethernet":
		if l.Mbps > 0 {
			return "ethernet " + speedLabel(l.Mbps), l.Mbps
		}
		return "ethernet", 0
	default:
		return l.Kind, l.Mbps
	}
}

// LocalAddrToward returns this machine's address on the interface that shares
// a subnet with peer — the address peers should use to reach us — or the first
// non-loopback IPv4 address when nothing matches.
func LocalAddrToward(peer string) string {
	if ip := net.ParseIP(peer); ip != nil {
		if _, own := interfaceToward(ip); own != nil {
			return own.String()
		}
	}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() && n.IP.To4() != nil {
			return n.IP.String()
		}
	}
	return ""
}

// interfaceToward finds the local interface whose subnet contains ip, and our
// own address on it.
func interfaceToward(ip net.IP) (string, net.IP) {
	ifs, err := net.Interfaces()
	if err != nil {
		return "", nil
	}
	for _, i := range ifs {
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.Contains(ip) {
				return i.Name, n.IP
			}
		}
	}
	return "", nil
}

func speedLabel(mbps int) string {
	switch {
	case mbps >= 1000 && mbps%1000 == 0:
		return strconv.Itoa(mbps/1000) + "G"
	case mbps >= 1000:
		return fmt.Sprintf("%.1fG", float64(mbps)/1000)
	default:
		return strconv.Itoa(mbps) + "M"
	}
}
