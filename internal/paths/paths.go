// Package paths resolves the ~/.connex directory and the files inside it.
// CONNEX_HOME overrides the location (used by tests and power users).
package paths

import (
	"os"
	"path/filepath"
)

// Dir returns the connex home directory, creating it 0700 if missing.
func Dir() (string, error) {
	dir := os.Getenv("CONNEX_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".connex")
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
