package discovery

import (
	"net"
	"testing"
)

func TestParsePeer(t *testing.T) {
	if p, ok := ParsePeer("169.254.1.2:5000"); !ok || p.Addr != "169.254.1.2" || p.Port != 5000 {
		t.Fatalf("good peer failed: %v %v", p, ok)
	}
	for _, bad := range []string{"", ":5000", "1.2.3.4:0", "1.2.3.4:70000", "host:abc"} {
		if _, ok := ParsePeer(bad); ok {
			t.Fatalf("accepted bad peer %q", bad)
		}
	}
}

func TestParsePeerList(t *testing.T) {
	got := ParsePeerList([]string{" 1.2.3.4:1 ", "", "bad:port", "5.6.7.8:2"})
	if len(got) != 2 || got[0].Port != 1 || got[1].Port != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestLinkLabels(t *testing.T) {
	if label, _ := Link("127.0.0.1"); label != "local" {
		t.Fatalf("loopback: %q", label)
	}
	if label, _ := Link("203.0.113.9"); label != "net" { // TEST-NET, never on a local subnet
		t.Fatalf("routed: %q", label)
	}
	if label, _ := Link("not-an-ip"); label != "" {
		t.Fatalf("garbage: %q", label)
	}
}

func TestFastestPrefersHigherSpeed(t *testing.T) {
	link := func(a string) (string, int) {
		return "", map[string]int{"169.254.1.2": 40000, "192.168.1.9": 0}[a]
	}
	ips := []net.IP{net.ParseIP("192.168.1.9"), net.ParseIP("169.254.1.2")}
	if got := fastest(ips, link); got != "169.254.1.2" {
		t.Fatalf("got %q", got)
	}
	if got := fastest(nil, link); got != "" {
		t.Fatalf("empty: %q", got)
	}
}

func TestLocalAddrTowardSelf(t *testing.T) {
	// Our own address is on the interface toward itself.
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() && n.IP.To4() != nil {
			if got := LocalAddrToward(n.IP.String()); got != n.IP.String() {
				t.Fatalf("toward %s: got %s", n.IP, got)
			}
			return
		}
	}
	t.Skip("no non-loopback IPv4 interface")
}

func TestParsePeerDefaultsPort(t *testing.T) {
	p, ok := ParsePeer("studio.tailnet.ts.net")
	if !ok || p.Addr != "studio.tailnet.ts.net" || p.Port != DefaultPort {
		t.Fatalf("got %+v ok=%v, want DefaultPort %d", p, ok, DefaultPort)
	}
	if p, ok := ParsePeer("10.0.0.7:7000"); !ok || p.Port != 7000 {
		t.Fatalf("explicit port lost: %+v ok=%v", p, ok)
	}
	if _, ok := ParsePeer(""); ok {
		t.Fatal("empty string parsed as a peer")
	}
}
