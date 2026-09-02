//go:build !windows

// Package agent is the server half of knit: it advertises this machine over
// mDNS, wraps every connection in TLS 1.3, authenticates it with an HMAC over
// a fresh nonce bound to that connection, proves its own key back, and serves
// the info and run operations. See docs/02-architecture.md.
package agent

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/oddurs/knit/internal/discovery"
	"github.com/oddurs/knit/internal/keys"
	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/sysinfo"
	"github.com/oddurs/knit/internal/transport"
)

// authTimeout bounds how long a client has to complete the handshake.
const authTimeout = 10 * time.Second

// Serve runs the agent in the foreground until signaled. It loads (or creates)
// the cluster key, listens, advertises over mDNS, and handles connections until
// SIGINT/SIGTERM.
func Serve() error {
	key, err := keys.LoadOrCreate()
	if err != nil {
		return err
	}
	cfg, err := TLSConfig()
	if err != nil {
		return err
	}
	ln, err := listen()
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

	log.Printf("knit: %s up on port %d — %d cpus, %.1f GB", info.Name, port, info.CPUs, info.MemGB)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	stopping := make(chan struct{})
	go func() {
		<-sig
		close(stopping)
		_ = ln.Close()
	}()
	// The key is re-read per connection so `knit key --rotate` takes effect
	// without restarting the agent; the startup key is the fallback.
	loadKey := func() []byte {
		if k, err := keys.LoadOrCreate(); err == nil {
			return k
		}
		return key
	}
	return serve(ln, cfg, loadKey, stopping)
}

// listen prefers the well-known port so `--peer host` works across restarts,
// and falls back to an ephemeral port (mDNS carries either) if it is taken.
func listen() (net.Listener, error) {
	if ln, err := net.Listen("tcp", ":"+strconv.Itoa(discovery.DefaultPort)); err == nil {
		return ln, nil
	}
	return net.Listen("tcp", ":0")
}

// serve accepts connections until stopping is closed (and ln with it). Other
// accept errors — typically a transient fd shortage — are logged and retried
// after a short pause rather than spun on.
func serve(ln net.Listener, cfg *tls.Config, loadKey func() []byte, stopping <-chan struct{}) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-stopping:
				return nil
			default:
			}
			log.Printf("knit: accept: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go handleConn(conn, cfg, loadKey)
	}
}

// TLSConfig builds the agent's TLS configuration around a fresh self-signed
// certificate. The certificate is ephemeral and never verified by clients;
// it only keys the channel. Authentication is the key-bound proof exchange.
func TLSConfig() (*tls.Config, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: priv}},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// Handle serves one accepted connection under a fixed key. Tests in other
// packages use it to stand up a real agent on loopback.
func Handle(conn net.Conn, cfg *tls.Config, key []byte) {
	handleConn(conn, cfg, func() []byte { return key })
}

func handleConn(raw net.Conn, cfg *tls.Config, loadKey func() []byte) {
	defer raw.Close()
	_ = raw.SetDeadline(time.Now().Add(authTimeout))
	conn := tls.Server(raw, cfg)
	if err := conn.Handshake(); err != nil {
		return
	}
	cb, err := transport.ChannelBinding(conn)
	if err != nil {
		return
	}

	nonce, err := keys.Nonce(16)
	if err != nil {
		return
	}
	if _, err := fmt.Fprintf(conn, "%s %s\n", proto.Magic, nonce); err != nil {
		return
	}
	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return
	}
	var req proto.Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return
	}
	key := loadKey()
	if !keys.Equal(keys.ClientProof(key, nonce, cb), req.HMAC) {
		writeEnvelope(conn, proto.Envelope{
			Code:  proto.CodeUnauthorized,
			Error: "unauthorized: run `knit key` on a trusted machine and `knit join <key>` here",
		})
		return
	}
	_ = conn.SetDeadline(time.Time{})
	proof := keys.ServerProof(key, nonce, cb)

	// The connection is authenticated once. An info op may be followed by a
	// run on the same connection, so a `knit run` reuses the probe it just did
	// (KN-XPORT-050); run and dial each consume the connection and return.
	for {
		switch req.Op {
		case proto.OpInfo:
			env := sysinfo.Local()
			env.Proof = proof
			writeEnvelope(conn, env)
			next, ok := readFollowup(conn, br)
			if !ok {
				return
			}
			req = next
		case proto.OpRun:
			handleRun(conn, br, req, proof)
			return
		case proto.OpDial:
			handleDial(conn, req, proof)
			return
		default:
			writeEnvelope(conn, proto.Envelope{Code: proto.CodeInternal, Error: "unknown op"})
			return
		}
	}
}

// reuseIdle bounds how long an authenticated connection waits for a follow-up
// op after an info reply before closing, so a held probe never leaks.
const reuseIdle = 10 * time.Second

func readFollowup(conn net.Conn, br *bufio.Reader) (proto.Request, bool) {
	_ = conn.SetReadDeadline(time.Now().Add(reuseIdle))
	line, err := br.ReadString('\n')
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		return proto.Request{}, false
	}
	var req proto.Request
	if json.Unmarshal([]byte(line), &req) != nil {
		return proto.Request{}, false
	}
	return req, true
}

func writeEnvelope(w io.Writer, e proto.Envelope) {
	b, _ := json.Marshal(e)
	_, _ = w.Write(append(b, '\n'))
}
