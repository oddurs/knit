// Package keys implements knit authentication: a shared 32-byte cluster key
// and HMAC-SHA256 proofs bound to the TLS connection they are sent over. The
// key never crosses the wire; a passive listener learns nothing reusable, and
// a machine in the middle cannot forward a proof because it is tied to the
// channel binding of the connection it was computed for. See
// docs/08-security-model.md and docs/adr/0009-tls-key-bound-handshake.md.
package keys

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oddurs/knit/internal/paths"
)

// KeyLen is the cluster key length in bytes.
const KeyLen = 32

// LoadOrCreate returns the cluster key, generating and persisting one 0600 on
// first use.
func LoadOrCreate() ([]byte, error) {
	path, err := paths.KeyFile()
	if err != nil {
		return nil, err
	}
	if b, err := os.ReadFile(path); err == nil {
		return decode(strings.TrimSpace(string(b)), path)
	}
	return Rotate()
}

// Rotate generates a fresh key and installs it atomically, so no reader ever
// sees a partial or missing key. It returns the new key.
func Rotate() ([]byte, error) {
	key := make([]byte, KeyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, install(key)
}

// Save installs a key provided as 64 hex characters (from `knit key`).
func Save(hexKey string) error {
	key, err := decode(strings.TrimSpace(hexKey), "input")
	if err != nil {
		return err
	}
	return install(key)
}

// install writes the key to a temporary file beside the keyfile and renames it
// into place.
func install(key []byte) error {
	path, err := paths.KeyFile()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".key-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(hex.EncodeToString(key) + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Print returns the current key as 64 hex characters, creating it if needed.
func Print() (string, error) {
	key, err := LoadOrCreate()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func decode(s, where string) ([]byte, error) {
	key, err := hex.DecodeString(s)
	if err != nil || len(key) != KeyLen {
		return nil, fmt.Errorf("key must be %d hex characters (%s)", KeyLen*2, where)
	}
	return key, nil
}

// ClientProof is what a client sends to prove it holds key on the connection
// with channel binding cb, for the server's nonce.
func ClientProof(key []byte, nonce string, cb []byte) string {
	return proof(key, "knit client", nonce, cb)
}

// ServerProof is what a server sends back to prove it holds the same key on
// the same connection, so the client knows it is not talking to an impostor.
func ServerProof(key []byte, nonce string, cb []byte) string {
	return proof(key, "knit server", nonce, cb)
}

func proof(key []byte, label, nonce string, cb []byte) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(label))
	m.Write([]byte{0})
	m.Write([]byte(nonce))
	m.Write([]byte{0})
	m.Write(cb)
	return hex.EncodeToString(m.Sum(nil))
}

// Equal compares two proofs in constant time.
func Equal(want, got string) bool { return hmac.Equal([]byte(want), []byte(got)) }

// Nonce returns n random bytes hex-encoded (2n characters).
func Nonce(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
