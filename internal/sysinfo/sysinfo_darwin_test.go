//go:build darwin

package sysinfo

import "testing"

func TestParseVMLoadavg(t *testing.T) {
	if got := parseVMLoadavg("{ 1.85 2.07 2.10 }\n"); got != 1.85 {
		t.Fatalf("got %v", got)
	}
	if got := parseVMLoadavg("{ }"); got != 0 {
		t.Fatalf("empty: got %v", got)
	}
}

const vmStatFixture = `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                   273462.
Pages active:                                1283090.
Pages inactive:                              1243525.
Pages speculative:                             44246.
Pages wired down:                             192306.
Pages purgeable:                               35127.
File-backed pages:                            826748.
Anonymous pages:                             1744113.
Pages occupied by compressor:                  47913.
`

func TestParseVMStatFreeGB(t *testing.T) {
	// used = (192306 + 47913 + 1744113 - 35127) pages * 16 KiB = 29.7 GiB of 48.
	got := parseVMStatFreeGB(vmStatFixture, 48.0)
	if got < 18.2 || got > 18.4 {
		t.Fatalf("got %v, want ~18.3", got)
	}
	if got := parseVMStatFreeGB("garbage", 48.0); got != 0 {
		t.Fatalf("garbage: got %v", got)
	}
}

func TestParseHardwarePorts(t *testing.T) {
	in := `
Hardware Port: Thunderbolt Bridge
Device: bridge0
Ethernet Address: 36:45:32:36:80:00

Hardware Port: Wi-Fi
Device: en0
Ethernet Address: c0:c7:db:25:b8:a1

Hardware Port: Thunderbolt 1
Device: en1
Ethernet Address: 36:45:32:36:80:00
`
	m := parseHardwarePorts(in)
	if m["bridge0"] != "Thunderbolt Bridge" || m["en0"] != "Wi-Fi" || m["en1"] != "Thunderbolt 1" {
		t.Fatalf("got %v", m)
	}
}

func TestParseIfconfigMbps(t *testing.T) {
	in := "en5: flags=8863<UP> mtu 1500\n\tmedia: autoselect (10Gbase-T <full-duplex>)\n\tstatus: active\n"
	if got := parseIfconfigMbps(in); got != 10000 {
		t.Fatalf("got %d", got)
	}
}
