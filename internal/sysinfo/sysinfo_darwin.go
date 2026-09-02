//go:build darwin

package sysinfo

import (
	"os/exec"
	"strconv"
	"strings"
)

func load1() float64 {
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return 0
	}
	return parseVMLoadavg(string(out))
}

// parseVMLoadavg parses sysctl vm.loadavg, e.g. "{ 1.85 2.07 2.10 }".
func parseVMLoadavg(s string) float64 {
	f := strings.Fields(strings.Trim(strings.TrimSpace(s), "{}"))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}

func totalMemGB() float64 {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	bytes, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	return roundGB(bytes)
}
