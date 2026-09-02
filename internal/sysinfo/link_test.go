package sysinfo

import "testing"

func TestMediaMbps(t *testing.T) {
	cases := map[string]int{
		"media: autoselect (1000baseT <full-duplex>)": 1000,
		"10Gbase-T":         10000,
		"2.5GBase-T":        2500,
		"100baseTX":         100,
		"media: autoselect": 0,
		"":                  0,
	}
	for in, want := range cases {
		if got := mediaMbps(in); got != want {
			t.Fatalf("mediaMbps(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestLocalReportsAccelAndFree(t *testing.T) {
	e := Local()
	if e.Accel == "" {
		t.Fatalf("accel not populated: %+v", e)
	}
	if e.MemFreeGB < 0 || e.MemFreeGB > e.MemGB+0.1 {
		t.Fatalf("implausible free memory: %+v", e)
	}
}
