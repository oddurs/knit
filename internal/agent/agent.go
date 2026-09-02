// Package agent is the server half of connex: it advertises this machine over
// mDNS, authenticates each connection with an HMAC over a fresh nonce, and
// serves the info and run operations. See docs/02-architecture.md.
package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/oddurs/connex/internal/discovery"
	"github.com/oddurs/connex/internal/keys"
	"github.com/oddurs/connex/internal/proto"
	"github.com/oddurs/connex/internal/sysinfo"
)

// authTimeout bounds how long a client has to complete the handshake.
const authTimeout = 10 * time.Second

// Serve runs the agent in the foreground until signaled. It loads (or creates)
// the cluster key, listens on an ephemeral port, advertises over mDNS, and
// handles connections until SIGINT/SIGTERM.
func Serve() error {
	key, err := keys.LoadOrCreate()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	port := ln.Addr().(*net.TCPAddr).Port

	info := sysinfo.Local()
	txt := []string{
		"v=" + strconv.Itoa(proto.Version),
		"os=" + runtime.GOOS,
		"arch=" + runtime.GOARCH,
		"cpus=" + strconv.Itoa(info.CPUs),
	}
	server, err := discovery.Register(info.Name, port, txt)
	if err != nil {
		return fmt.Errorf("mDNS registration failed: %w", err)
	}
	defer server.Shutdown()

	log.Printf("connex agent: %s advertising on port %d (%d cpus, %.1f GB)",
		info.Name, port, info.CPUs, info.MemGB)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		server.Shutdown()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-sig:
				return nil // clean shutdown
			default:
				continue
			}
		}
		go handleConn(conn, key)
	}
}

func handleConn(conn net.Conn, key []byte) {
	defer conn.Close()

	nonce, err := keys.Nonce(16)
	if err != nil {
		return
	}
	if _, err := fmt.Fprintf(conn, "%s %s\n", proto.Magic, nonce); err != nil {
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return
	}
	var req proto.Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return
	}
	if !keys.Verify(key, nonce, req.HMAC) {
		writeEnvelope(conn, proto.Envelope{
			Code:  proto.CodeUnauthorized,
			Error: "unauthorized: run `connex key` on a trusted machine and `connex join <key>` here",
		})
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	switch req.Op {
	case proto.OpInfo:
		writeEnvelope(conn, sysinfo.Local())
	case proto.OpRun:
		handleRun(conn, br, req)
	default:
		writeEnvelope(conn, proto.Envelope{Code: proto.CodeInternal, Error: "unknown op"})
	}
}

func writeEnvelope(w io.Writer, e proto.Envelope) {
	b, _ := json.Marshal(e)
	_, _ = w.Write(append(b, '\n'))
}
