// Package transport dials an agent and performs the CONNEX1 client handshake:
// read the nonce, prove knowledge of the cluster key with an HMAC, send the
// request, and read the server's reply envelope. TCP_NODELAY is set on connect
// because connex streams many small frames interactively. See docs/03-protocol.md.
package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/oddurs/connex/internal/keys"
	"github.com/oddurs/connex/internal/proto"
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

// Open dials addr, completes the handshake for req under key, and returns the
// session. It sets a deadline around the handshake and clears it before return
// so streaming is not time-bounded.
func Open(addr string, key []byte, req proto.Request, dialTimeout time.Duration) (*Session, error) {
	conn, err := Dial(addr, dialTimeout)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(HandshakeTimeout)); err != nil {
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
	req.HMAC = keys.Sign(key, nonce)
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
	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, err
	}
	return &Session{Conn: conn, R: r, Reply: reply}, nil
}

func parseGreeting(line string) (string, error) {
	f := strings.Fields(strings.TrimSpace(line))
	if len(f) != 2 || f[0] != proto.Magic {
		return "", fmt.Errorf("not a connex agent (greeting %q)", strings.TrimSpace(line))
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
