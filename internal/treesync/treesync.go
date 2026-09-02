// Package treesync moves a working directory between machines for `knit run
// --dir`. It streams a directory as a tar (never buffering the whole tree in
// memory), unpacks it safely, and — for --sync — computes which files changed
// during a run by content hash so only those are mirrored back. See
// docs/02-architecture.md and KN-EXEC-030.
package treesync

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteTar streams the regular files and directories under root into w as a tar
// archive, using paths relative to root. Symlinks and other special files are
// skipped in v0.2. It does not buffer the whole tree; each file is copied
// through in turn.
func WriteTar(w io.Writer, root string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			hdr := &tar.Header{Name: rel + "/", Mode: int64(info.Mode().Perm()), Typeflag: tar.TypeDir}
			return tw.WriteHeader(hdr)
		case info.Mode().IsRegular():
			hdr := &tar.Header{Name: rel, Mode: int64(info.Mode().Perm()), Size: info.Size(), Typeflag: tar.TypeReg}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		default:
			return nil // skip symlinks, devices, sockets in v0.2
		}
	})
}

// ReadTar unpacks a tar stream from r into root, creating directories as needed.
// It rejects any entry whose path escapes root (path traversal defense).
func ReadTar(r io.Reader, root string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		dest, err := safeJoin(root, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil { //nolint:gosec // size bounded by sender's disk
				f.Close()
				return err
			}
			f.Close()
		}
	}
}

// safeJoin joins root and a tar entry name, rejecting anything that would escape
// root via "..", an absolute path, or a symlink-style trick.
func safeJoin(root, name string) (string, error) {
	clean := filepath.Clean("/" + name) // makes it rooted, collapses ..
	dest := filepath.Join(root, clean)
	if dest != root && !strings.HasPrefix(dest, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe path in archive: %q", name)
	}
	return dest, nil
}

// Snapshot returns a map of relative path -> content hash for every regular file
// under root. It is taken before a run so ChangedSince can find what the command
// modified.
func Snapshot(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		h, err := hashFile(path)
		if err != nil {
			return err
		}
		out[rel] = h
		return nil
	})
	return out, err
}

// ChangedSince returns, sorted, the relative paths of regular files under root
// that are new or whose content differs from the snapshot — a content-hash
// delta. Deletions are not mirrored in v0.2.
func ChangedSince(root string, snap map[string]string) ([]string, error) {
	var changed []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		h, err := hashFile(path)
		if err != nil {
			return err
		}
		if old, ok := snap[rel]; !ok || old != h {
			changed = append(changed, rel)
		}
		return nil
	})
	sort.Strings(changed)
	return changed, err
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteTarFiles streams only the named relative files (a delta) under root into
// w as a tar. Used for the --sync mirror-back.
func WriteTarFiles(w io.Writer, root string, rels []string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	for _, rel := range rels {
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		hdr := &tar.Header{Name: rel, Mode: int64(info.Mode().Perm()), Size: info.Size(), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}
