package keys

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProofsAreBoundToKeyNonceAndChannel(t *testing.T) {
	key := make([]byte, KeyLen)
	for i := range key {
		key[i] = byte(i)
	}
	cb := []byte("channel-binding-of-this-connection")
	c := ClientProof(key, "abc123", cb)
	if !Equal(c, ClientProof(key, "abc123", cb)) {
		t.Fatal("proof not deterministic")
	}
	if Equal(c, ServerProof(key, "abc123", cb)) {
		t.Fatal("client and server proofs must differ")
	}
	if Equal(c, ClientProof(key, "abc124", cb)) {
		t.Fatal("proof ignored the nonce")
	}
	if Equal(c, ClientProof(key, "abc123", []byte("another connection"))) {
		t.Fatal("proof ignored the channel binding — a relay would pass")
	}
	if Equal(c, ClientProof(make([]byte, KeyLen), "abc123", cb)) {
		t.Fatal("proof ignored the key")
	}
}

func TestRotateReplacesKeyAtomically(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIT_HOME", dir)
	k1, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	k2, err := Rotate()
	if err != nil {
		t.Fatal(err)
	}
	if string(k1) == string(k2) {
		t.Fatal("rotate kept the same key")
	}
	k3, _ := LoadOrCreate()
	if string(k3) != string(k2) {
		t.Fatal("rotated key not persisted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "key" {
		t.Fatalf("temp file left behind: %v", entries)
	}
	if info, _ := os.Stat(filepath.Join(dir, "key")); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode().Perm())
	}
}

func TestLoadOrCreateAndSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIT_HOME", dir)

	k1, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if len(k1) != KeyLen {
		t.Fatalf("key len = %d", len(k1))
	}
	// Persisted 0600.
	info, err := os.Stat(filepath.Join(dir, "key"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %v", info.Mode().Perm())
	}
	// Stable across reads.
	k2, err := LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if string(k1) != string(k2) {
		t.Fatal("key changed between loads")
	}
}

func TestSaveRejectsBadKey(t *testing.T) {
	t.Setenv("KNIT_HOME", t.TempDir())
	if err := Save("not-hex"); err == nil {
		t.Fatal("accepted non-hex key")
	}
	if err := Save("abcd"); err == nil {
		t.Fatal("accepted short key")
	}
}
