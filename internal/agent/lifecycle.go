package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/oddurs/knit/internal/paths"
	"github.com/oddurs/knit/internal/sysinfo"
)

// Mode is how `knit up` runs the agent.
type Mode int

const (
	Foreground Mode = iota // in this terminal, until Ctrl-C
	Detached               // re-exec in the background; `knit down` stops it
	Forever                // install an OS service that restarts it and survives login
)

// Up starts the agent in the given mode. Detached and Forever are idempotent:
// an agent that is already up is reported, not duplicated.
func Up(mode Mode) error {
	switch mode {
	case Detached:
		return daemonize()
	case Forever:
		return install()
	}
	return Serve()
}

func daemonize() error {
	if sysinfo.ServiceInstalled() {
		fmt.Printf("knit already up (managed by the %s)\n", sysinfo.ServiceName())
		return nil
	}
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

// install registers the agent with launchd or systemd so it starts at login,
// is restarted if it dies, and outlives this terminal. A detached agent that
// is already running is stopped first so there is one agent, not two.
func install() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	logPath, err := paths.LogFile()
	if err != nil {
		return err
	}
	if pid, ok := runningPid(); ok {
		_ = syscall.Kill(pid, syscall.SIGTERM)
		if p, err := paths.PidFile(); err == nil {
			_ = os.Remove(p)
		}
	}
	env := map[string]string{}
	if h := os.Getenv("KNIT_HOME"); h != "" {
		env["KNIT_HOME"] = h
	}
	if err := sysinfo.InstallService(self, logPath, env); err != nil {
		return err
	}
	fmt.Printf("knit up — installed as a %s; it starts at login and restarts if it stops\n", sysinfo.ServiceName())
	return nil
}

// Down stops the agent however it was started: removes the OS service if one
// is installed, else signals the detached agent. Nothing running is reported,
// not treated as a failure.
func Down() error {
	if sysinfo.ServiceInstalled() {
		if err := sysinfo.UninstallService(); err != nil {
			return fmt.Errorf("removing the %s: %w", sysinfo.ServiceName(), err)
		}
		fmt.Printf("knit down — removed the %s; this machine left the fabric\n", sysinfo.ServiceName())
		return nil
	}
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
