package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/oddurs/connex/internal/paths"
)

// Up starts the agent. When detach is true it re-execs itself in the background
// (setsid, logging to ~/.connex/agent.log, pidfile for Down); otherwise it runs
// in the foreground.
func Up(detach bool) error {
	if detach {
		return daemonize()
	}
	return Serve()
}

func daemonize() error {
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
	fmt.Printf("connex agent up (pid %d) — this machine is now sharing compute\n", c.Process.Pid)
	return nil
}

// Down stops the background agent recorded in the pidfile.
func Down() error {
	pidPath, err := paths.PidFile()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("no agent pidfile — is the agent running in the background?")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return fmt.Errorf("corrupt pidfile: %w", err)
	}
	proc, err := os.FindProcess(pid)
	if err == nil {
		err = proc.Signal(syscall.SIGTERM)
	}
	_ = os.Remove(pidPath)
	if err != nil {
		return fmt.Errorf("stopping agent: %w", err)
	}
	fmt.Println("connex agent stopped")
	return nil
}
