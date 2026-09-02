// Package paths resolves the ~/.knit directory and the files inside it.
// KNIT_HOME overrides the location (used by tests and power users).
package paths

import (
	"os"
	"path/filepath"

	"github.com/oddurs/knit/internal/proto"
)

// Dir returns the knit home directory, creating it 0700 if missing.
func Dir() (string, error) {
	dir := os.Getenv("KNIT_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".knit")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func file(name string) (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, name), nil
}

// KeyFile is the shared cluster key.
func KeyFile() (string, error) { return file("key") }

// PidFile records the detached agent's pid.
func PidFile() (string, error) { return file("agent.pid") }

// LogFile is the detached agent's log.
func LogFile() (string, error) { return file("agent.log") }

// PeersCache is the short-lived discovered-peer cache.
func PeersCache() (string, error) { return file("peers.json") }

// RankEnv writes the hostfile for a ranked launch into the knit directory and
// returns the environment to run the command with plus a cleanup that removes
// the file. Each launch gets its own file so concurrent launches never collide.
func RankEnv(rank int, hosts []string) (env []string, cleanup func(), err error) {
	d, err := Dir()
	if err != nil {
		return nil, nil, err
	}
	f, err := os.CreateTemp(d, "hostfile-*.json")
	if err != nil {
		return nil, nil, err
	}
	if _, err := f.Write(proto.Hostfile(hosts)); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, nil, err
	}
	f.Close()
	return proto.RankEnv(rank, hosts, f.Name()), func() { os.Remove(f.Name()) }, nil
}
