package scheduler

import (
	"testing"

	"github.com/oddurs/knit/internal/proto"
)

func cand(name string, cpus int, load float64, local bool) Candidate {
	return Candidate{Name: name, Local: local, Info: proto.Envelope{CPUs: cpus, Load1: load}}
}

func TestPickLowestLoadPerCore(t *testing.T) {
	cands := []Candidate{
		cand("here", 8, 4.0, true),     // 0.50
		cand("studio", 24, 0.3, false), // 0.0125
		cand("mini", 10, 1.85, false),  // 0.185
	}
	best, ok := Pick(cands)
	if !ok || best.Name != "studio" {
		t.Fatalf("picked %q ok=%v", best.Name, ok)
	}
}

func TestPickPrefersLocalOnTie(t *testing.T) {
	cands := []Candidate{
		cand("here", 8, 0.8, true),
		cand("peer", 8, 0.8, false),
	}
	best, _ := Pick(cands)
	if best.Name != "here" {
		t.Fatalf("tie should prefer local first element, got %q", best.Name)
	}
}

func TestPickEmpty(t *testing.T) {
	if _, ok := Pick(nil); ok {
		t.Fatal("empty set should return ok=false")
	}
}

func TestByName(t *testing.T) {
	cands := []Candidate{cand("here", 8, 1, true), cand("studio", 24, 1, false)}
	if c, ok := ByName(cands, "studio"); !ok || c.Name != "studio" {
		t.Fatalf("ByName failed: %v %v", c, ok)
	}
	if _, ok := ByName(cands, "nope"); ok {
		t.Fatal("ByName found nonexistent")
	}
}

func TestPickTieBreaksTowardFasterLink(t *testing.T) {
	cands := []Candidate{
		cand("here", 8, 8.0, true), // 1.0, busy
		{Name: "wifi", Info: proto.Envelope{CPUs: 8, Load1: 0.80}, LinkMbps: 0},
		{Name: "cable", Info: proto.Envelope{CPUs: 8, Load1: 0.84}, LinkMbps: 40000}, // within tie
	}
	if best, _ := Pick(cands); best.Name != "cable" {
		t.Fatalf("picked %q, want cable (faster link within tie)", best.Name)
	}
	cands[2].Info.Load1 = 1.2 // clearly busier: speed must not override load
	if best, _ := Pick(cands); best.Name != "wifi" {
		t.Fatalf("picked %q, want wifi", best.Name)
	}
}

func TestFilterMemAndArch(t *testing.T) {
	cands := []Candidate{
		{Name: "here", Local: true, Info: proto.Envelope{Arch: "arm64", MemFreeGB: 12}},
		{Name: "studio", Info: proto.Envelope{Arch: "arm64", MemFreeGB: 90}},
		{Name: "box", Info: proto.Envelope{Arch: "amd64", MemFreeGB: 200}},
	}
	got, why := Filter(cands, 48, "")
	if why != "" || len(got) != 2 || got[0].Name != "studio" {
		t.Fatalf("mem: got %v why=%q", got, why)
	}
	got, why = Filter(cands, 48, "arm64")
	if why != "" || len(got) != 1 || got[0].Name != "studio" {
		t.Fatalf("mem+arch: got %v why=%q", got, why)
	}
	if _, why = Filter(cands, 500, ""); why != "no machine has 500 GB free (most: box with 200.0 GB)" {
		t.Fatalf("mem too large: %q", why)
	}
	if _, why = Filter(cands, 0, "riscv64"); why != "no riscv64 machine reachable" {
		t.Fatalf("arch: %q", why)
	}
	if got, _ := Filter(cands, 0, ""); len(got) != 3 {
		t.Fatalf("no constraint should keep all: %v", got)
	}
}
