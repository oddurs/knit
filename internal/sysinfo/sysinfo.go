// Package sysinfo reports this machine's capacity: cores, total memory, and
// 1-minute load. Free-memory and accelerator fields arrive in v0.3 (KN-SYS-030).
// The OS-specific probes live in sysinfo_darwin.go and sysinfo_linux.go.
package sysinfo

import (
	"os"
	"runtime"
	"strings"

	"github.com/oddurs/knit/internal/proto"
)

// Local returns a populated info envelope for this machine.
func Local() proto.Envelope {
	host, _ := os.Hostname()
	return proto.Envelope{
		OK:    true,
		Name:  ShortHost(host),
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
		CPUs:  runtime.NumCPU(),
		MemGB: totalMemGB(),
		Load1: load1(),
	}
}

// ShortHost trims a hostname to its leading label without the .local suffix.
func ShortHost(h string) string {
	h = strings.TrimSuffix(h, ".local")
	if i := strings.Index(h, "."); i > 0 {
		h = h[:i]
	}
	return h
}

// roundGB converts bytes to gibibytes rounded to one decimal place.
func roundGB(bytes float64) float64 {
	gb := bytes / (1 << 30)
	return float64(int(gb*10+0.5)) / 10
}
