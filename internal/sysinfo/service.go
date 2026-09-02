package sysinfo

// Service is the OS-managed supervisor for `knit up --forever`: a launchd
// agent on macOS, a systemd user unit on Linux. The functions here are the
// only OS-specific part of the agent's lifecycle.

// ServiceName is how the OS refers to the unit, for messages.
func ServiceName() string { return serviceName }

// InstallService writes the unit that runs `exe up` at login, restarts it if it
// dies, logs to logPath, and starts it now. env is extra environment for the
// unit (KNIT_HOME when set).
func InstallService(exe, logPath string, env map[string]string) error {
	return installService(exe, logPath, env)
}

// ServiceInstalled reports whether the unit exists.
func ServiceInstalled() bool { return serviceInstalled() }

// UninstallService stops the unit and removes it.
func UninstallService() error { return uninstallService() }
