//go:build linux

package sysinfo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const serviceName = "systemd user unit knit.service"

func unitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		home = dir
	} else {
		home = filepath.Join(home, ".config")
	}
	return filepath.Join(home, "systemd", "user", "knit.service"), nil
}

func installService(exe, logPath string, env map[string]string) error {
	path, err := unitPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var envLines strings.Builder
	for k, v := range env {
		fmt.Fprintf(&envLines, "Environment=%s=%s\n", k, v)
	}
	unit := fmt.Sprintf(`[Unit]
Description=knit agent — share this machine's compute
After=network.target

[Service]
ExecStart=%s up
Restart=always
RestartSec=2
StandardOutput=append:%s
StandardError=append:%s
%s
[Install]
WantedBy=default.target
`, exe, logPath, logPath, envLines.String())
	if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
		return err
	}
	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", "--now", "knit.service"},
	} {
		if out, err := exec.Command("systemctl", append([]string{"--user"}, args...)...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl --user %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	// Without lingering, user units stop at logout and do not start at boot
	// until the user logs in. Best effort: it may need a password prompt.
	_ = exec.Command("loginctl", "enable-linger").Run()
	return nil
}

func serviceInstalled() bool {
	path, err := unitPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func uninstallService() error {
	path, err := unitPath()
	if err != nil {
		return err
	}
	_ = exec.Command("systemctl", "--user", "disable", "--now", "knit.service").Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}
