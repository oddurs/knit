//go:build linux

package sysinfo

import (
	"os"
	"os/exec"
	"path/filepath"
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

func totalMemGB() float64 { return meminfoGB("MemTotal:") }

// freeMemGB is the kernel's own estimate of allocatable memory without
// swapping (MemAvailable), which already accounts for reclaimable cache.
func freeMemGB() float64 { return meminfoGB("MemAvailable:") }

func meminfoGB(key string) float64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	return parseMeminfoGB(string(b), key)
}

func parseMeminfoGB(s, key string) float64 {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, key) {
			f := strings.Fields(line)
			if len(f) >= 2 {
				kb, _ := strconv.ParseFloat(f[1], 64)
				return roundGB(kb * 1024)
			}
		}
	}
	return 0
}

// probeAccel asks nvidia-smi, when present, for the first GPU's name and
// memory. Without an NVIDIA driver there is no accelerator knit can name.
func probeAccel() accel {
	smi, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return accel{accel: "none"}
	}
	out, err := exec.Command(smi, "--query-gpu=name,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return accel{accel: "none"}
	}
	gpu := parseNvidiaSMI(string(out))
	if gpu == "" {
		return accel{accel: "none"}
	}
	return accel{gpu: gpu, accel: "cuda"}
}

// parseNvidiaSMI turns "NVIDIA GeForce RTX 4090, 24564" into "NVIDIA GeForce
// RTX 4090 24G".
func parseNvidiaSMI(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	name, mem, ok := strings.Cut(line, ",")
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if !ok {
		return name
	}
	mib, err := strconv.ParseFloat(strings.TrimSpace(mem), 64)
	if err != nil || mib <= 0 {
		return name
	}
	return name + " " + strconv.Itoa(int(mib/1024+0.5)) + "G"
}

// linkOf classifies an interface from sysfs: a wireless directory means Wi-Fi,
// a thunderbolt name means IP over Thunderbolt, a physical device with a
// reported speed is Ethernet. Virtual interfaces return the zero Link.
func linkOf(iface string) Link { return linkFromSysfs(filepath.Join("/sys/class/net", iface), iface) }

func linkFromSysfs(dir, iface string) Link {
	if _, err := os.Stat(filepath.Join(dir, "wireless")); err == nil {
		return Link{Kind: "wifi"}
	}
	if strings.HasPrefix(iface, "thunderbolt") {
		return Link{Kind: "thunderbolt", Mbps: 40000}
	}
	if _, err := os.Stat(filepath.Join(dir, "device")); err != nil {
		return Link{} // no backing device: bridge, veth, tunnel
	}
	b, _ := os.ReadFile(filepath.Join(dir, "speed"))
	mbps, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if mbps < 0 {
		mbps = 0
	}
	return Link{Kind: "ethernet", Mbps: mbps}
}
