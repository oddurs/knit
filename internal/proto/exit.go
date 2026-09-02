package proto

import (
	"os/exec"
	"syscall"
)

// ExitStatus maps an exec.Cmd Run/Wait result to the code a local shell would
// report: the process's own status, 128+signal when a signal ended it, or 127
// when the program could not be started at all. Both halves of knit use it so
// a command's exit code is the same whether it ran here or on a peer.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		return 127
	}
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return ee.ExitCode()
}
