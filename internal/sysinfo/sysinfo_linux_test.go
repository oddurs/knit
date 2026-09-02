//go:build linux

package sysinfo

import "testing"

func TestParseLoadavg(t *testing.T) {
	if got := parseLoadavg("0.42 0.35 0.30 1/234 5678\n"); got != 0.42 {
		t.Fatalf("got %v", got)
	}
	if got := parseLoadavg(""); got != 0 {
		t.Fatalf("empty: got %v", got)
	}
}

func TestParseMemTotalGB(t *testing.T) {
	in := "MemFree: 100 kB\nMemTotal:   16384000 kB\nBuffers: 1 kB\n"
	if got := parseMemTotalGB(in); got < 15.5 || got > 15.7 {
		t.Fatalf("got %v", got)
	}
	if got := parseMemTotalGB("no total here\n"); got != 0 {
		t.Fatalf("missing: got %v", got)
	}
}
