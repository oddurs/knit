package sysinfo

import "testing"

func TestShortHost(t *testing.T) {
	cases := map[string]string{
		"studio.local":     "studio",
		"mini.lan.example": "mini",
		"host":             "host",
	}
	for in, want := range cases {
		if got := ShortHost(in); got != want {
			t.Fatalf("ShortHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRoundGB(t *testing.T) {
	if got := roundGB(16 * (1 << 30)); got != 16.0 {
		t.Fatalf("got %v", got)
	}
}

func TestLocalPopulated(t *testing.T) {
	e := Local()
	if !e.OK || e.CPUs < 1 || e.OS == "" || e.Arch == "" {
		t.Fatalf("implausible local info: %+v", e)
	}
}
