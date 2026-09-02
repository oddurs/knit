package client

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/oddurs/connex/internal/proto"
	"github.com/oddurs/connex/internal/sysinfo"
)

// LS lists this machine and every reachable, authorized agent with live
// capacity. It always browses fresh (not cached) so the picture is current.
func LS(jsonOut bool) int {
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "connex:", err)
		return 1
	}
	cands := probePeers(key, true)

	self := sysinfo.Local()
	self.Self = true

	if jsonOut {
		out := []proto.Envelope{self}
		for _, c := range cands {
			out = append(out, c.Info)
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return 0
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].Name < cands[j].Name })
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tADDR\tOS/ARCH\tCPUS\tMEM\tLOAD")
	for _, c := range cands {
		i := c.Info
		fmt.Fprintf(tw, "%s\t%s\t%s/%s\t%d\t%.1fG\t%.2f\n",
			i.Name, c.Addr, i.OS, i.Arch, i.CPUs, i.MemGB, i.Load1)
	}
	fmt.Fprintf(tw, "%s\t%s\t%s/%s\t%d\t%.1fG\t%.2f  (this machine)\n",
		self.Name, "—", self.OS, self.Arch, self.CPUs, self.MemGB, self.Load1)
	tw.Flush()
	return 0
}
