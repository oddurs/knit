package client

import (
	"net"
	"strings"
	"testing"

	"github.com/oddurs/knit/internal/agent"
	"github.com/oddurs/knit/internal/sysinfo"
)

func clientTestKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 3)
	}
	return k
}

// fakeInfoAgent is a real agent handler on a loopback listener under the given
// key. It advertises nothing over mDNS, so it is only reachable as an explicit
// peer. It reports this machine's real name and capacity.
func fakeInfoAgent(t *testing.T, key []byte) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	cfg, err := agent.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go agent.Handle(c, cfg, key)
		}
	}()
	return ln.Addr().String()
}

// TestExplicitPeerDiscoveredViaEnv verifies KN-DISC-020: a peer given in
// KNIT_PEERS (not advertised over mDNS) is probed and scheduled, with its link
// classified from the address.
func TestExplicitPeerDiscoveredViaEnv(t *testing.T) {
	extraPeers = nil // isolate from other tests
	t.Setenv("KNIT_HOME", t.TempDir())
	t.Setenv("CONNEX_HOME", t.TempDir()) // belt and suspenders
	key := clientTestKey()
	addr := fakeInfoAgent(t, key)
	t.Setenv("KNIT_PEERS", addr)

	cands, _ := probePeers(key, false)
	var found *string
	for i := range cands {
		if cands[i].Name == sysinfo.Name() {
			n := cands[i].Info.Link
			found = &n
			if cands[i].Info.CPUs < 1 {
				t.Fatalf("wrong cpus: %d", cands[i].Info.CPUs)
			}
		}
	}
	if found == nil {
		t.Fatal("explicit KNIT_PEERS agent was not discovered")
	}
	if *found != "local" { // 127.0.0.1 is loopback
		t.Fatalf("link = %q, want local", *found)
	}
}

// TestExplicitPeerRejectsWrongKey confirms an explicit peer with a key mismatch
// is simply dropped, never scheduled.
func TestExplicitPeerRejectsWrongKey(t *testing.T) {
	extraPeers = nil
	t.Setenv("KNIT_HOME", t.TempDir())
	addr := fakeInfoAgent(t, clientTestKey())
	t.Setenv("KNIT_PEERS", addr)

	wrong := make([]byte, 32) // all zeros
	cands, fails := probePeers(wrong, false)
	if len(cands) != 0 {
		t.Fatalf("unauthorized peer was scheduled: %v", cands)
	}
	if len(fails) != 1 || !strings.Contains(fails[0].Err.Error(), "unauthorized") {
		t.Fatalf("explicit peer failure not reported: %v", fails)
	}
}

// TestOlderAgentReported: an explicitly named peer that still speaks plaintext
// (knit ≤ v0.3) is reported as such, so a mixed-version fabric explains itself.
func TestOlderAgentReported(t *testing.T) {
	extraPeers = nil
	t.Setenv("KNIT_HOME", t.TempDir())
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); c.Write([]byte("KNIT1 00\n")); c.Read(make([]byte, 1)) }()
		}
	}()
	t.Setenv("KNIT_PEERS", ln.Addr().String())
	cands, fails := probePeers(clientTestKey(), false)
	if len(cands) != 0 || len(fails) != 1 || !strings.Contains(fails[0].Err.Error(), "older knit") {
		t.Fatalf("cands=%v fails=%v", cands, fails)
	}
}
