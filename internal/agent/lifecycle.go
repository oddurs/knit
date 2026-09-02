package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/oddurs/knit/internal/paths"
)

// Up starts the agent. When detach is true it re-execs itself in the background
// (setsid, logging to ~/.knit/agent.log, pidfile for Down); otherwise it runs
// in the foreground. A second `knit up -d` while the agent is running is a
// no-op that reports the existing pid, so repeating it is always safe.
func Up(detach bool) error {
	if detach {
		return daemonize()
	}
	return Serve()
}

func daemonize() error {
	if pid, ok := runningPid(); ok {
		fmt.Printf("knit already up (pid %d)\n", pid)
		return nil
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	logPath, err := paths.LogFile()
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()

	c := exec.Command(self, "up")
	c.Stdout = logf
	c.Stderr = logf
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		return err
	}
	pidPath, err := paths.PidFile()
	if err != nil {
		return err
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(c.Process.Pid)+"\n"), 0o600); err != nil {
		return err
	}
	fmt.Printf("knit up (pid %d) — this machine is now a loop in the fabric\n", c.Process.Pid)
	return nil
}

// Down stops the background agent recorded in the pidfile. A stale pidfile
// (agent already gone) is cleaned up and reported, not treated as a failure.
func Down() error {
	pidPath, err := paths.PidFile()
	if err != nil {
		return err
	}
	pid, ok := runningPid()
	_ = os.Remove(pidPath)
	if !ok {
		fmt.Println("knit down — no background agent was running")
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("stopping agent: %w", err)
	}
	fmt.Println("knit down — this machine left the fabric")
	return nil
}

// runningPid reads the pidfile and reports the pid only if that process is
// still alive.
func runningPid() (int, bool) {
	pidPath, err := paths.PidFile()
	if err != nil {
		return 0, false
	}
	b, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return 0, false
	}
	return pid, true
}
