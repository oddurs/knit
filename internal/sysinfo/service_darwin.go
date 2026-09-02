//go:build darwin

package sysinfo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const serviceName = "launchd agent io.knit.agent"

const label = "io.knit.agent"

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

func domain() string { return "gui/" + strconv.Itoa(os.Getuid()) }

func installService(exe, logPath string, env map[string]string) error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, plist(exe, logPath, env), 0o644); err != nil {
		return err
	}
	// A previous registration (or a running -d agent's leftovers) must not
	// block the bootstrap; bootout is a no-op when nothing is loaded.
	_ = exec.Command("launchctl", "bootout", domain()+"/"+label).Run()
	if out, err := exec.Command("launchctl", "bootstrap", domain(), path).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %s", firstLine(out, err))
	}
	return nil
}

func serviceInstalled() bool {
	path, err := plistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func uninstallService() error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	_ = exec.Command("launchctl", "bootout", domain()+"/"+label).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// plist renders the launchd property list: run at login, keep alive, log to
// the agent log.
func plist(exe, logPath string, env map[string]string) []byte {
	var b []byte
	b = append(b, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>`+label+`</string>
  <key>ProgramArguments</key><array><string>`+esc(exe)+`</string><string>up</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>`+esc(logPath)+`</string>
  <key>StandardErrorPath</key><string>`+esc(logPath)+`</string>
`...)
	if len(env) > 0 {
		b = append(b, "  <key>EnvironmentVariables</key><dict>\n"...)
		for k, v := range env {
			b = append(b, "    <key>"+esc(k)+"</key><string>"+esc(v)+"</string>\n"...)
		}
		b = append(b, "  </dict>\n"...)
	}
	return append(b, "</dict></plist>\n"...)
}

var esc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace

func firstLine(out []byte, err error) string {
	for i, c := range out {
		if c == '\n' {
			out = out[:i]
			break
		}
	}
	if len(out) == 0 {
		return err.Error()
	}
	return string(out)
}
