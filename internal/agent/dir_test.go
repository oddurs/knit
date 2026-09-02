package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/transport"
	"github.com/oddurs/knit/internal/treesync"
)

// TestRunDirSync verifies KN-EXEC-030 end to end at the agent: a working dir is
// streamed in, the command runs there and reads a sent file, and --sync mirrors
// the changed/new files back by content hash.
func TestRunDirSync(t *testing.T) {
	// Build an input tree: a.txt only.
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	var inTar bytes.Buffer
	if err := treesync.WriteTar(&inTar, src); err != nil {
		t.Fatal(err)
	}

	addr := serveOne(t, testKey())
	sess, err := transport.Open(addr, testKey(),
		proto.Request{Op: proto.OpRun, Dir: true, Sync: true,
			Cmd: []string{"sh", "-c", "printf %s \"$(cat a.txt)\"; printf NEW > b.txt"}},
		2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	_ = sess.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send the input tree, then an empty stdin.
	fw := proto.NewFrameWriter(sess.Conn)
	if err := fw.Write(proto.FrameInTar, inTar.Bytes()); err != nil {
		t.Fatal(err)
	}
	fw.Write(proto.FrameInEnd, nil)
	fw.Write(proto.FrameStdinEOF, nil)

	// Collect stdout and the mirror-back tar.
	var out, outTar bytes.Buffer
	code := -1
	for code < 0 {
		typ, p, err := proto.ReadFrame(sess.R)
		if err != nil {
			t.Fatalf("read frame: %v", err)
		}
		switch typ {
		case proto.FrameStdout:
			out.Write(p)
		case proto.FrameOutTar:
			outTar.Write(p)
		case proto.FrameExit:
			code = proto.ExitCode(p)
		}
	}
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "hello" {
		t.Fatalf("command did not read sent file: stdout=%q", out.String())
	}

	// Apply the mirror-back and confirm only b.txt came home with the right content.
	dst := t.TempDir()
	if err := treesync.ReadTar(&outTar, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "b.txt"))
	if err != nil || strings.TrimSpace(string(b)) != "NEW" {
		t.Fatalf("b.txt not mirrored back correctly: %q err=%v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "a.txt")); !os.IsNotExist(err) {
		t.Fatal("unchanged a.txt should not be in the sync-back delta")
	}
}
