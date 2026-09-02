//go:build windows

// The knit agent needs Unix process-group control (setpgid, killpg) to run and
// reap remote commands, which Windows lacks; only the client half builds here.
// These stubs let cmd/knit compile so `knit run`, `gauge`, `each`, and the key
// commands work on Windows, while `knit up`/`down` report the agent is
// unsupported.
package agent

import "errors"

// Mode is accepted for parity with the Unix build; no mode runs on Windows.
type Mode int

const (
	Foreground Mode = iota
	Detached
	Forever
)

var errUnsupported = errors.New("the knit agent is not supported on Windows; run `knit up` on a macOS or Linux machine and use this one as a client")

// Up reports that the agent cannot run on Windows.
func Up(Mode) error { return errUnsupported }

// Down reports that there is no agent to stop on Windows.
func Down() error { return errUnsupported }
