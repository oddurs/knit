package proto

import "os/exec"

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
	if code, ok := signalExit(ee); ok {
		return code
	}
	return ee.ExitCode()
}
