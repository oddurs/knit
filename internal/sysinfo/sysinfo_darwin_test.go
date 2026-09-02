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
