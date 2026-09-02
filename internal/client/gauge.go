package client

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/oddurs/knit/internal/proto"
	"github.com/oddurs/knit/internal/sysinfo"
)

// Gauge lists this machine and every reachable, authorized machine with live
// capacity. Gauge is the knitting word for a capacity measurement, which is what
// this is. It always browses fresh (not cached) so the picture is current.
func Gauge(jsonOut bool) int {
	key, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
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
	fmt.Fprintln(tw, "NAME\tADDR\tOS/ARCH\tCPUS\tMEM\tFREE\tLOAD\tGPU\tLINK")
	for _, c := range cands {
		i := c.Info
		fmt.Fprintf(tw, "%s\t%s\t%s/%s\t%d\t%.1fG\t%.1fG\t%.2f\t%s\t%s\n",
			i.Name, c.Addr, i.OS, i.Arch, i.CPUs, i.MemGB, i.MemFreeGB, i.Load1, gpuCell(i), i.Link)
	}
	fmt.Fprintf(tw, "%s\t%s\t%s/%s\t%d\t%.1fG\t%.1fG\t%.2f\t%s\t%s\n",
		self.Name, "—", self.OS, self.Arch, self.CPUs, self.MemGB, self.MemFreeGB, self.Load1, gpuCell(self), "(this machine)")
	tw.Flush()
	return 0
}

// gpuCell is the GPU column: the accelerator's name, or the backend when the
// name is unknown, or a dash when there is none.
func gpuCell(e proto.Envelope) string {
	switch {
	case e.GPU != "":
		return e.GPU
	case e.Accel != "" && e.Accel != "none":
		return e.Accel
	}
	return "—"
}
