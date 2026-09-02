package treesync

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTarRoundTrip(t *testing.T) {
	src := t.TempDir()
	write(t, src, "a.txt", "alpha")
	write(t, src, "sub/b.txt", "beta")
	write(t, src, "sub/deep/c.txt", "gamma")

	var buf bytes.Buffer
	if err := WriteTar(&buf, src); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := ReadTar(&buf, dst); err != nil {
		t.Fatal(err)
	}
	if read(t, dst, "a.txt") != "alpha" || read(t, dst, "sub/b.txt") != "beta" || read(t, dst, "sub/deep/c.txt") != "gamma" {
		t.Fatal("tar round-trip corrupted content")
	}
}

func TestSafeJoinContainsTraversal(t *testing.T) {
	// A "../" entry must be neutralized to stay inside root, never escape it.
	root := "/root"
	got, err := safeJoin(root, "../etc/passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/root/etc/passwd" {
		t.Fatalf("traversal not contained: got %q", got)
	}
	// A deep "../../.." attempt is likewise pinned under root.
	got, _ = safeJoin(root, "../../../../etc/shadow")
	if got != "/root/etc/shadow" {
		t.Fatalf("traversal not contained: got %q", got)
	}
	// Normal nested paths are joined as-is.
	got, _ = safeJoin(root, "sub/ok")
	if got != "/root/sub/ok" {
		t.Fatalf("safe path mangled: got %q", got)
	}
}

func TestChangedSince(t *testing.T) {
	root := t.TempDir()
	write(t, root, "keep.txt", "same")
	write(t, root, "edit.txt", "before")
	snap, err := Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	// Modify one file, add a new one, leave keep.txt alone.
	write(t, root, "edit.txt", "after")
	write(t, root, "new.txt", "fresh")

	changed, err := ChangedSince(root, snap)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"edit.txt": true, "new.txt": true}
	if len(changed) != 2 || !want[changed[0]] || !want[changed[1]] {
		t.Fatalf("changed = %v, want edit.txt+new.txt only", changed)
	}
}

func TestWriteTarFilesDelta(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "A")
	write(t, root, "b.txt", "B")
	var buf bytes.Buffer
	if err := WriteTarFiles(&buf, root, []string{"b.txt"}); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := ReadTar(&buf, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "a.txt")); !os.IsNotExist(err) {
		t.Fatal("delta tar should not include a.txt")
	}
	if read(t, dst, "b.txt") != "B" {
		t.Fatal("delta tar missing b.txt")
	}
}
