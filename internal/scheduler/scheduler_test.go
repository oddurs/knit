package scheduler

import (
	"testing"

	"github.com/oddurs/connex/internal/proto"
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
