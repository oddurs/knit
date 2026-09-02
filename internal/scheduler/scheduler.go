// Package scheduler picks where a command runs. v1 is deliberately dumb: score
// each candidate as load-per-core and take the lowest, with the local machine
// always a candidate. See docs/adr/0006-load-per-core-scheduler.md.
package scheduler

import (
	"net"
	"strconv"

	"github.com/oddurs/knit/internal/proto"
)

// Candidate is a machine the scheduler may place work on. Local candidates have
// an empty Addr and are executed in-process rather than dialed.
type Candidate struct {
	Name  string
	Addr  string
	Port  int
	Info  proto.Envelope
	Local bool
	// LinkMbps is the nominal speed of the link to this machine, 0 when
	// unknown; the local machine is treated as infinitely fast.
	LinkMbps int
}

// tie is how close two scores must be (in load per core) to count as equal.
const tie = 0.01

// Pick returns the lowest-scoring candidate. The bool is false when the set is
// empty. Scores within tie of each other are broken toward the faster link;
// the caller orders the slice with the local machine first, and local counts
// as the fastest link, so a true tie prefers staying local.
func Pick(cands []Candidate) (Candidate, bool) {
	if len(cands) == 0 {
		return Candidate{}, false
	}
	best := cands[0]
	bestScore := best.Info.Score()
	for _, c := range cands[1:] {
		s := c.Info.Score()
		switch {
		case s < bestScore-tie:
			best, bestScore = c, s
		case s <= bestScore+tie && c.linkMbps() > best.linkMbps():
			best, bestScore = c, s
		}
	}
	return best, true
}

func (c Candidate) linkMbps() int {
	if c.Local {
		return 1 << 30
	}
	return c.LinkMbps
}

// Filter keeps the candidates that satisfy the placement constraints: at least
// memGB free (0 = any) and, when arch is set, that architecture. The string
// names the constraint that emptied the set, for the error the user sees.
func Filter(cands []Candidate, memGB float64, arch string) ([]Candidate, string) {
	var out []Candidate
	bestName, bestFree := "", -1.0
	for _, c := range cands {
		if arch != "" && c.Info.Arch != arch {
			continue
		}
		if c.Info.MemFreeGB > bestFree {
			bestName, bestFree = c.Name, c.Info.MemFreeGB
		}
		if memGB > 0 && c.Info.MemFreeGB < memGB {
			continue
		}
		out = append(out, c)
	}
	if len(out) > 0 {
		return out, ""
	}
	if bestName == "" {
		return nil, "no " + arch + " machine reachable"
	}
	return nil, "no machine has " + strconv.FormatFloat(memGB, 'f', -1, 64) +
		" GB free (most: " + bestName + " with " + strconv.FormatFloat(bestFree, 'f', 1, 64) + " GB)"
}

// ByName returns the candidate with the given name, for --on pinning.
func ByName(cands []Candidate, name string) (Candidate, bool) {
	for _, c := range cands {
		if c.Name == name {
			return c, true
		}
	}
	return Candidate{}, false
}

// HostPortOrEmpty returns the dial target for a remote candidate, or "" if it
// is the local machine.
func (c Candidate) HostPortOrEmpty() string {
	if c.Local || c.Addr == "" {
		return ""
	}
	return net.JoinHostPort(c.Addr, strconv.Itoa(c.Port))
}
