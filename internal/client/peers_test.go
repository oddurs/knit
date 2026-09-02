package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/oddurs/knit/internal/keys"
	"github.com/oddurs/knit/internal/proto"
)

func clientTestKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i * 3)
	}
	return k
}

// fakeInfoAgent is a minimal agent double: it completes the handshake and, for a
// valid HMAC, replies with a fixed info envelope. It advertises nothing over
// mDNS, so it is only reachable as an explicit peer.
func fakeInfoAgent(t *testing.T, key []byte, name string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				nonce, _ := keys.Nonce(16)
				fmt.Fprintf(c, "%s %s\n", proto.Magic, nonce)
				line, err := bufio.NewReader(c).ReadString('\n')
				if err != nil {
					return
				}
				var req proto.Request
				if json.Unmarshal([]byte(line), &req) != nil {
					return
				}
				var env proto.Envelope
				if keys.Verify(key, nonce, req.HMAC) {
					env = proto.Envelope{OK: true, Name: name, OS: "linux", Arch: "arm64", CPUs: 4, MemGB: 8, Load1: 0.1}
				} else {
					env = proto.Envelope{Code: proto.CodeUnauthorized, Error: "no"}
				}
				b, _ := json.Marshal(env)
				c.Write(append(b, '\n'))
			}(c)
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
	addr := fakeInfoAgent(t, key, "studio")
	t.Setenv("KNIT_PEERS", addr)

	cands := probePeers(key, false)
	var found *string
	for i := range cands {
		if cands[i].Name == "studio" {
			n := cands[i].Info.Link
			found = &n
			if cands[i].Info.CPUs != 4 {
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
	addr := fakeInfoAgent(t, clientTestKey(), "studio")
	t.Setenv("KNIT_PEERS", addr)

	wrong := make([]byte, 32) // all zeros
	if cands := probePeers(wrong, false); len(cands) != 0 {
		t.Fatalf("unauthorized peer was scheduled: %v", cands)
	}
}
