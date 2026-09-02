package transport_test

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/oddurs/knit/internal/agent"
	"github.com/oddurs/knit/internal/keys"
	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/transport"
)

func key() []byte { return []byte("0123456789abcdef0123456789abcdef") }

// TestOlderAgentNamed: an agent that greets in plaintext (knit ≤ v0.3) is
// reported as such, with the fix, not as a TLS failure.
func TestOlderAgentNamed(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		fmt.Fprintf(c, "%s %s\n", proto.Magic, "00")
		_, _ = bufio.NewReader(c).ReadString('\n')
	}()
	_, err := transport.Open(ln.Addr().String(), key(), proto.Request{Op: proto.OpInfo}, time.Second)
	re, ok := err.(*transport.ReplyError)
	if !ok || re.Code != proto.CodeVersion || !strings.Contains(re.Msg, "older knit") {
		t.Fatalf("got %v", err)
	}
}

// TestImpostorAgentRejected: a server that accepts the client's proof but
// cannot produce its own (it does not hold the key) is refused, so a machine
// in the middle or a stale agent with another key never receives a command.
func TestImpostorAgentRejected(t *testing.T) {
	cfg, _ := agent.TLSConfig()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		raw, err := ln.Accept()
		if err != nil {
			return
		}
		c := tls.Server(raw, cfg)
		defer c.Close()
		if err := c.Handshake(); err != nil {
			return
		}
		fmt.Fprintf(c, "%s %s\n", proto.Magic, "00")
		_, _ = bufio.NewReader(c).ReadString('\n')
		b, _ := json.Marshal(proto.Envelope{OK: true, Name: "evil", Proof: keys.ServerProof([]byte("wrong-key-wrong-key-wrong-key-00"), "00", nil)})
		_, _ = c.Write(append(b, '\n'))
	}()
	_, err := transport.Open(ln.Addr().String(), key(), proto.Request{Op: proto.OpInfo}, time.Second)
	re, ok := err.(*transport.ReplyError)
	if !ok || re.Code != proto.CodeUnauthorized {
		t.Fatalf("impostor accepted: %v", err)
	}
}
