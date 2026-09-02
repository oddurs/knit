//go:build linux

package sysinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLoadavg(t *testing.T) {
	if got := parseLoadavg("0.42 0.35 0.30 1/234 5678\n"); got != 0.42 {
		t.Fatalf("got %v", got)
	}
	if got := parseLoadavg(""); got != 0 {
		t.Fatalf("empty: got %v", got)
	}
}

func TestParseMeminfoGB(t *testing.T) {
	in := "MemFree: 100 kB\nMemTotal:   16384000 kB\nMemAvailable: 8192000 kB\nBuffers: 1 kB\n"
	if got := parseMeminfoGB(in, "MemTotal:"); got < 15.5 || got > 15.7 {
		t.Fatalf("got %v", got)
	}
	if got := parseMeminfoGB(in, "MemAvailable:"); got < 7.7 || got > 7.9 {
		t.Fatalf("available: got %v", got)
	}
	if got := parseMeminfoGB("no total here\n", "MemTotal:"); got != 0 {
		t.Fatalf("missing: got %v", got)
	}
}

func TestParseNvidiaSMI(t *testing.T) {
	if got := parseNvidiaSMI("NVIDIA GeForce RTX 4090, 24564\n"); got != "NVIDIA GeForce RTX 4090 24G" {
		t.Fatalf("got %q", got)
	}
	if got := parseNvidiaSMI(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestLinkFromSysfs(t *testing.T) {
	dir := t.TempDir()
	if got := linkFromSysfs(dir, "veth0"); got.Kind != "" {
		t.Fatalf("virtual: got %+v", got)
	}
	os.Mkdir(filepath.Join(dir, "wireless"), 0o755)
	if got := linkFromSysfs(dir, "wlan0"); got.Kind != "wifi" {
		t.Fatalf("wifi: got %+v", got)
	}
	eth := t.TempDir()
	os.Mkdir(filepath.Join(eth, "device"), 0o755)
	os.WriteFile(filepath.Join(eth, "speed"), []byte("2500\n"), 0o644)
	if got := linkFromSysfs(eth, "eth0"); got.Kind != "ethernet" || got.Mbps != 2500 {
		t.Fatalf("ethernet: got %+v", got)
	}
	if got := linkFromSysfs(t.TempDir(), "thunderbolt0"); got.Kind != "thunderbolt" {
		t.Fatalf("thunderbolt: got %+v", got)
	}
}
