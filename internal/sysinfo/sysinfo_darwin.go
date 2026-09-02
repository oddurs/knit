//go:build darwin

package sysinfo

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func sysctl(name string) string {
	out, err := exec.Command("sysctl", "-n", name).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func load1() float64 { return parseVMLoadavg(sysctl("vm.loadavg")) }

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
	bytes, _ := strconv.ParseFloat(sysctl("hw.memsize"), 64)
	return roundGB(bytes)
}

// freeMemGB is what could be allocated now without paging anything out,
// computed the way Activity Monitor does: total minus (wired + compressed +
// app memory), where app memory is anonymous pages less purgeable ones. Inactive
// file cache counts as free, which is what a model load actually gets.
func freeMemGB() float64 {
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0
	}
	return parseVMStatFreeGB(string(out), totalMemGB())
}

func parseVMStatFreeGB(s string, totalGB float64) float64 {
	pageSize := 4096.0
	pages := map[string]float64{}
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "Mach Virtual Memory Statistics") {
			if i := strings.Index(line, "page size of "); i >= 0 {
				f := strings.Fields(line[i+len("page size of "):])
				if len(f) > 0 {
					if v, err := strconv.ParseFloat(f[0], 64); err == nil {
						pageSize = v
					}
				}
			}
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(v), "."), 64)
		if err != nil {
			continue
		}
		pages[strings.TrimSpace(k)] = n
	}
	used := pages["Pages wired down"] + pages["Pages occupied by compressor"] +
		pages["Anonymous pages"] - pages["Pages purgeable"]
	free := totalGB - used*pageSize/(1<<30)
	if free < 0 || used == 0 {
		return 0
	}
	return roundGB(free * (1 << 30))
}

// probeAccel: every Mac has Metal; on Apple silicon the GPU is the chip and
// its memory is the unified pool already reported as mem_gb.
func probeAccel() accel {
	a := accel{accel: "metal"}
	if runtime.GOARCH == "arm64" {
		a.gpu = sysctl("machdep.cpu.brand_string")
	}
	return a
}

var (
	portsOnce sync.Once
	ports     map[string]string // device (en0) -> hardware port name ("Wi-Fi")
)

// linkOf classifies an interface from `networksetup -listallhardwareports`,
// read once per process, with Ethernet speed taken from the ifconfig media line.
func linkOf(iface string) Link {
	portsOnce.Do(func() {
		out, _ := exec.Command("networksetup", "-listallhardwareports").Output()
		ports = parseHardwarePorts(string(out))
	})
	port := ports[iface]
	switch {
	case strings.HasPrefix(port, "Thunderbolt"):
		return Link{Kind: "thunderbolt", Mbps: 40000}
	case port == "Wi-Fi":
		return Link{Kind: "wifi"}
	case strings.Contains(port, "Ethernet") || strings.Contains(port, "LAN"):
		out, _ := exec.Command("ifconfig", iface).Output()
		return Link{Kind: "ethernet", Mbps: parseIfconfigMbps(string(out))}
	}
	return Link{}
}

// parseHardwarePorts maps devices to port names from networksetup output.
func parseHardwarePorts(s string) map[string]string {
	m := map[string]string{}
	port := ""
	for _, line := range strings.Split(s, "\n") {
		if v, ok := strings.CutPrefix(line, "Hardware Port: "); ok {
			port = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(line, "Device: "); ok && port != "" {
			m[strings.TrimSpace(v)] = port
		}
	}
	return m
}

// parseIfconfigMbps reads the negotiated speed from an ifconfig media line,
// e.g. "media: autoselect (1000baseT <full-duplex>)".
func parseIfconfigMbps(s string) int {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "media:") {
			return mediaMbps(line)
		}
	}
	return 0
}
