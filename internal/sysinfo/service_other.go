//go:build !linux && !darwin

package sysinfo

import "errors"

const serviceName = "service"

func installService(string, string, map[string]string) error {
	return errors.New("knit up --forever is not supported on this OS")
}
func serviceInstalled() bool  { return false }
func uninstallService() error { return nil }
