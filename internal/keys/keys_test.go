package keys

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	key := make([]byte, KeyLen)
	for i := range key {
		key[i] = byte(i)
	}
	sig := Sign(key, "abc123")
	if !Verify(key, "abc123", sig) {
		t.Fatal("valid signature rejected")
	}
	if Verify(key, "abc124", sig) {
		t.Fatal("signature verified against wrong nonce")
	}
	other := make([]byte, KeyLen)
	if Verify(other, "abc123", sig) {
		t.Fatal("signature verified under wrong key")
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
