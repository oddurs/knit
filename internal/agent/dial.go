//go:build !windows

package agent

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/oddurs/knit/internal/proto"
)

// handleDial opens a raw TCP tunnel to req.Host and splices bytes both ways
// over the (already TLS-encrypted, already authenticated) connection. It backs
// `knit proxy`: the peer reaches the network, the client rides its link. Only a
// key holder gets here, so this is arbitrary outbound connection on the peer's
// behalf — the same trust `run` already grants.
func handleDial(conn net.Conn, req proto.Request, proof string) {
	if req.Host == "" {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeInternal, Error: "dial needs a host"})
		return
	}
	target, err := net.DialTimeout("tcp", req.Host, 10*time.Second)
	if err != nil {
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeSpawn, Error: err.Error()})
		return
	}
	defer target.Close()
	writeEnvelope(conn, proto.Envelope{OK: true, Proof: proof})
	log.Printf("dial: %s", req.Host)

	done := make(chan struct{}, 2)
	splice := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		_ = dst.Close()
		_ = src.Close()
		done <- struct{}{}
	}
	go splice(target, conn)
	go splice(conn, target)
	<-done
	<-done
}
