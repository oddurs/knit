//go:build windows

package proto

import "os/exec"

// signalExit: Windows has no signal-death status; the exit code stands.
func signalExit(*exec.ExitError) (int, bool) { return 0, false }
