//go:build !windows

package proto

import (
	"os/exec"
	"syscall"
)

// signalExit reports 128+signal when a Unix process was ended by a signal.
func signalExit(ee *exec.ExitError) (int, bool) {
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal()), true
	}
	return 0, false
}
