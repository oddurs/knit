//go:build linux

package sysinfo

import (
	"os"
	"strconv"
	"strings"
)

func load1() float64 {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	return parseLoadavg(string(b))
}

func parseLoadavg(s string) float64 {
	f := strings.Fields(s)
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}

func totalMemGB() float64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	return parseMemTotalGB(string(b))
}

func parseMemTotalGB(s string) float64 {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				kb, _ := strconv.ParseFloat(f[1], 64)
				return roundGB(kb * 1024)
			}
		}
	}
	return 0
}
