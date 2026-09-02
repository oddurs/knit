// Package scheduler picks where a command runs. v1 is deliberately dumb: score
// each candidate as load-per-core and take the lowest, with the local machine
// always a candidate. See docs/adr/0006-load-per-core-scheduler.md.
package scheduler

import (
	"net"
	"strconv"

	"github.com/oddurs/connex/internal/proto"
)

// Candidate is a machine the scheduler may place work on. Local candidates have
// an empty Addr and are executed in-process rather than dialed.
type Candidate struct {
	Name  string
	Addr  string
	Port  int
	Info  proto.Envelope
	Local bool
}

// Pick returns the lowest-scoring candidate. The bool is false when the set is
// empty. Ties keep the earlier candidate, and the caller orders the slice with
// the local machine first so a tie prefers staying local.
func Pick(cands []Candidate) (Candidate, bool) {
	if len(cands) == 0 {
		return Candidate{}, false
	}
	best := cands[0]
	bestScore := best.Info.Score()
	for _, c := range cands[1:] {
		if s := c.Info.Score(); s < bestScore {
			best, bestScore = c, s
		}
	}
	return best, true
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
