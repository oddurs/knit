// Package paths resolves the ~/.knit directory and the files inside it.
// KNIT_HOME overrides the location (used by tests and power users).
package paths

import (
	"os"
	"path/filepath"
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
