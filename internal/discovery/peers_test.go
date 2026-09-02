package discovery

import (
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

func TestClassifyLink(t *testing.T) {
	cases := map[string]string{
		"169.254.87.3": "thunderbolt",
		"fe80::1":      "thunderbolt",
		"192.168.1.40": "lan",
		"10.0.0.5":     "lan",
		"127.0.0.1":    "local",
		"8.8.8.8":      "net",
		"not-an-ip":    "",
	}
	for addr, want := range cases {
		if got := ClassifyLink(addr); got != want {
			t.Fatalf("ClassifyLink(%q) = %q, want %q", addr, got, want)
		}
	}
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
