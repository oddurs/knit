// Package sysinfo reports this machine's capacity: cores, total and free
// memory, 1-minute load, and the accelerator, plus what kind of link a local
// network interface is. The OS-specific probes live in sysinfo_darwin.go and
// sysinfo_linux.go; everything here is portable.
package sysinfo

import (
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/oddurs/knit/internal/proto"
)

// Name is this machine's short hostname, as advertised over mDNS.
func Name() string {
	host, _ := os.Hostname()
	return ShortHost(host)
}

// Local returns a populated info envelope for this machine. Load and free
// memory are read live; the accelerator is probed once per process.
func Local() proto.Envelope {
	a := accelerator()
	return proto.Envelope{
		OK:        true,
		Name:      Name(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		MemGB:     totalMemGB(),
		MemFreeGB: freeMemGB(),
		Load1:     load1(),
		GPU:       a.gpu,
		Accel:     a.accel,
	}
}

// accel is the static accelerator description: what launchers pick a backend
// from. It never changes while the agent runs, so it is probed once.
type accel struct {
	gpu   string // human name, e.g. "Apple M5 Pro" or "NVIDIA RTX 4090 24G"
	accel string // metal | cuda | none
}

var (
	accelOnce sync.Once
	accelVal  accel
)

func accelerator() accel {
	accelOnce.Do(func() { accelVal = probeAccel() })
	return accelVal
}

// Link describes a local network interface for scheduling and display.
type Link struct {
	Kind string // thunderbolt | ethernet | wifi | "" (unknown or virtual)
	Mbps int    // nominal, 0 when unknown
}

// LinkOf reports what kind of link the named local interface is. It is a
// best-effort classification from the OS's own description of the interface;
// an interface the OS cannot describe (bridges, tunnels, VMs) returns the zero
// Link.
func LinkOf(iface string) Link { return linkOf(iface) }

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

// mediaMbps parses a link speed out of an interface media/speed description
// such as "1000baseT", "10Gbase-T", "2.5GBase-T", or "100baseTX".
func mediaMbps(s string) int {
	m := mediaRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	if m[2] != "" {
		v *= 1000
	}
	return int(v)
}

var mediaRe = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)(g)?base`)
