// Package transport dials an agent and performs the client handshake: a TLS
// 1.3 connection first, then over it the KNIT1 exchange — read the nonce,
// prove knowledge of the cluster key with an HMAC bound to this connection,
// send the request, read the reply, and check the server's own proof. The
// server's certificate is ephemeral and never verified; the key-bound proofs
// are what authenticate both ends. TCP_NODELAY is set because knit streams
// many small frames interactively. See docs/03-protocol.md.
package transport

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/oddurs/knit/internal/keys"
	"github.com/oddurs/knit/internal/proto"
)

// HandshakeTimeout bounds the control exchange before streaming begins.
const HandshakeTimeout = 10 * time.Second

// Session is an open, authenticated connection with the server's reply already
// read. For op "info" the reply carries capacity. For op "run" the reply is the
// ack and the caller proceeds to stream over Conn/R. The caller owns Close.
type Session struct {
	Conn  net.Conn
	R     *bufio.Reader
	Reply proto.Envelope
}

// Close releases the connection.
func (s *Session) Close() error { return s.Conn.Close() }

// Dial opens a TCP connection with Nagle disabled.
func Dial(addr string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
	}
	return conn, nil
}

// clientTLS never verifies the certificate: authentication comes from the
// key-bound proofs, and the certificate only serves to key the channel.
var clientTLS = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS13}

// Open dials addr, completes the handshake for req under key, and returns the
// session. It sets a deadline around the handshake and clears it before return
// so streaming is not time-bounded.
func Open(addr string, key []byte, req proto.Request, dialTimeout time.Duration) (*Session, error) {
	raw, err := Dial(addr, dialTimeout)
	if err != nil {
		return nil, err
	}
	conn := tls.Client(raw, clientTLS)
	if err := conn.SetDeadline(time.Now().Add(HandshakeTimeout)); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.Handshake(); err != nil {
		conn.Close()
		var rh tls.RecordHeaderError
		if errors.As(err, &rh) {
			return nil, &ReplyError{Code: proto.CodeVersion, Msg: "runs an older knit that speaks plaintext — upgrade knit there"}
		}
		return nil, fmt.Errorf("tls: %w", err)
	}
	cb, err := ChannelBinding(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	r := bufio.NewReader(conn)

	greeting, err := r.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("reading greeting: %w", err)
	}
	nonce, err := parseGreeting(greeting)
	if err != nil {
		conn.Close()
		return nil, err
	}

	req.V = proto.Version
	req.HMAC = keys.ClientProof(key, nonce, cb)
	line, _ := json.Marshal(req)
	if _, err := conn.Write(append(line, '\n')); err != nil {
		conn.Close()
		return nil, err
	}

	replyLine, err := r.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("reading reply: %w", err)
	}
	var reply proto.Envelope
	if err := json.Unmarshal([]byte(replyLine), &reply); err != nil {
		conn.Close()
		return nil, fmt.Errorf("bad reply: %w", err)
	}
	if !reply.OK {
		conn.Close()
		return nil, &ReplyError{Code: reply.Code, Msg: reply.Error}
	}
	if !keys.Equal(keys.ServerProof(key, nonce, cb), reply.Proof) {
		conn.Close()
		return nil, &ReplyError{Code: proto.CodeUnauthorized,
			Msg: "did not prove it holds this key (a different key, or something in the path)"}
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Session{Conn: conn, R: r, Reply: reply}, nil
}

// ChannelBinding derives 32 bytes unique to this TLS connection from its
// keying material. Both ends compute the same value; a connection relayed by
// a machine in the middle has a different one on each leg.
func ChannelBinding(conn *tls.Conn) ([]byte, error) {
	cs := conn.ConnectionState()
	cb, err := cs.ExportKeyingMaterial("knit channel binding", nil, 32)
	if err != nil {
		return nil, fmt.Errorf("channel binding: %w", err)
	}
	return cb, nil
}

func parseGreeting(line string) (string, error) {
	f := strings.Fields(strings.TrimSpace(line))
	if len(f) != 2 || f[0] != proto.Magic {
		return "", fmt.Errorf("not a knit agent (greeting %q)", strings.TrimSpace(line))
	}
	return f[1], nil
}

// ReplyError is returned when the server refuses the request; Code is a stable
// proto error code the CLI maps to a fix.
type ReplyError struct {
	Code string
	Msg  string
}

func (e *ReplyError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return e.Code
}
