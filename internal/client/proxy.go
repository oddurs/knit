package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/scheduler"
	"github.com/oddurs/knit/internal/transport"
)

// Proxy runs a local SOCKS5 listener that tunnels every connection through a
// peer's agent, so this machine reaches the network the peer can (KN-NET-040).
// The peer is chosen by name (onName) or, when there is exactly one, by
// default. Each tunnel is a TLS-encrypted, key-authenticated `dial` op reusing
// the same machinery as `run`. It runs until interrupted.
func Proxy(onName string, port int) int {
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}
	peers, fails := probePeers(key, true)
	for _, f := range fails {
		fmt.Fprintf(os.Stderr, "%s\n", dim("knit: "+f.HostPort+": "+f.Err.Error()))
	}
	target, code := pickProxyPeer(peers, onName)
	if code != 0 {
		return code
	}

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "knit: cannot listen on port %d: %v\n", port, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "%s\n", dim(fmt.Sprintf("knit proxy: SOCKS5 on %s → %s (Ctrl-C to stop)", ln.Addr(), target.Name)))
	for {
		conn, err := ln.Accept()
		if err != nil {
			return 1
		}
		go serveSocks(conn, target, key)
	}
}

// pickProxyPeer resolves which peer to tunnel through: the named one, the only
// one, or an error asking the user to choose.
func pickProxyPeer(peers []scheduler.Candidate, onName string) (scheduler.Candidate, int) {
	if onName != "" {
		if c, ok := scheduler.ByName(peers, onName); ok {
			return c, 0
		}
		fmt.Fprintf(os.Stderr, "knit: no reachable machine named %q\n", onName)
		return scheduler.Candidate{}, ExitUnreachable
	}
	switch len(peers) {
	case 0:
		fmt.Fprintln(os.Stderr, "knit: no other machines found to proxy through")
		return scheduler.Candidate{}, ExitUnreachable
	case 1:
		return peers[0], 0
	default:
		names := make([]string, len(peers))
		for i, c := range peers {
			names[i] = c.Name
		}
		fmt.Fprintf(os.Stderr, "knit: several machines available; choose one with --on NAME (%v)\n", names)
		return scheduler.Candidate{}, ExitUsage
	}
}

// serveSocks handles one SOCKS5 client: negotiate no-auth, read the CONNECT
// target, open a knit dial tunnel to it through the peer, and splice bytes.
func serveSocks(client net.Conn, target scheduler.Candidate, key []byte) {
	defer client.Close()
	host, err := socksHandshake(client)
	if err != nil {
		return
	}
	sess, err := transport.Open(target.HostPortOrEmpty(), key, proto.Request{Op: proto.OpDial, Host: host}, dialTimeout)
	if err != nil {
		_ = socksReply(client, 0x01) // general failure
		return
	}
	defer sess.Close()
	if err := socksReply(client, 0x00); err != nil {
		return
	}
	done := make(chan struct{}, 2)
	splice := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		_ = dst.Close()
		_ = src.Close()
		done <- struct{}{}
	}
	go splice(sess.Conn, client)
	go splice(client, sess.Conn)
	<-done
	<-done
}

// socksHandshake performs the SOCKS5 method negotiation and reads a CONNECT
// request, returning the target "host:port". Only CONNECT with no auth is
// supported — enough for a browser or curl.
func socksHandshake(c net.Conn) (string, error) {
	buf := make([]byte, 262)
	if _, err := io.ReadFull(c, buf[:2]); err != nil {
		return "", err
	}
	if buf[0] != 0x05 {
		return "", fmt.Errorf("not socks5")
	}
	nmethods := int(buf[1])
	if _, err := io.ReadFull(c, buf[:nmethods]); err != nil {
		return "", err
	}
	if _, err := c.Write([]byte{0x05, 0x00}); err != nil { // no authentication
		return "", err
	}
	// Request: ver, cmd, rsv, atyp
	if _, err := io.ReadFull(c, buf[:4]); err != nil {
		return "", err
	}
	if buf[1] != 0x01 { // CONNECT only
		_ = socksReply(c, 0x07)
		return "", fmt.Errorf("unsupported command %d", buf[1])
	}
	var host string
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(c, buf[:4]); err != nil {
			return "", err
		}
		host = net.IP(buf[:4]).String()
	case 0x03: // domain
		if _, err := io.ReadFull(c, buf[:1]); err != nil {
			return "", err
		}
		n := int(buf[0])
		if _, err := io.ReadFull(c, buf[:n]); err != nil {
			return "", err
		}
		host = string(buf[:n])
	case 0x04: // IPv6
		if _, err := io.ReadFull(c, buf[:16]); err != nil {
			return "", err
		}
		host = net.IP(buf[:16]).String()
	default:
		_ = socksReply(c, 0x08)
		return "", fmt.Errorf("unsupported address type %d", buf[3])
	}
	if _, err := io.ReadFull(c, buf[:2]); err != nil {
		return "", err
	}
	return net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(buf[:2])))), nil
}

// socksReply sends a SOCKS5 reply with the given status and a zero bind addr.
func socksReply(c net.Conn, status byte) error {
	_, err := c.Write([]byte{0x05, status, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	return err
}
