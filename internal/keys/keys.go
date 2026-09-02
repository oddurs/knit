// Package keys implements knit authentication: a shared 32-byte cluster key
// and HMAC-SHA256 proof over a per-connection nonce. The key never crosses the
// wire; a passive listener learns nothing reusable. See docs/08-security-model.md.
package keys

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
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
	key := make([]byte, KeyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(key)+"\n"), 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

// Save installs a key provided as 64 hex characters (from `knit key`).
func Save(hexKey string) error {
	key, err := decode(strings.TrimSpace(hexKey), "input")
	if err != nil {
		return err
	}
	path, err := paths.KeyFile()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hex.EncodeToString(key)+"\n"), 0o600)
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

// Sign returns the hex HMAC-SHA256 of nonce under key.
func Sign(key []byte, nonce string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(nonce))
	return hex.EncodeToString(m.Sum(nil))
}

// Verify reports whether sig is a valid signature of nonce, in constant time.
func Verify(key []byte, nonce, sig string) bool {
	want := Sign(key, nonce)
	return hmac.Equal([]byte(want), []byte(sig))
}

// Nonce returns n random bytes hex-encoded (2n characters).
func Nonce(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
