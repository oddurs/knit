package proto

import (
	"encoding/json"
	"strconv"
	"strings"
)

// RankEnv is the environment a `knit each` launch exports to every process it
// starts, so multi-node launchers need no hand-written host list:
//
//	KNIT_RANK     this machine's position in the launch, 0 = the machine
//	              knit each was run on
//	KNIT_NNODES   number of machines in the launch
//	KNIT_HOSTS    comma-separated addresses in rank order
//	KNIT_MASTER   address of rank 0 (torchrun's --master_addr)
//	MLX_RANK      = KNIT_RANK, as MLX's ring backend expects
//	MLX_HOSTFILE  path of an MLX-format hostfile written for this launch
//
// hostfile is the path the caller wrote Hostfile(hosts) to.
func RankEnv(rank int, hosts []string, hostfile string) []string {
	master := ""
	if len(hosts) > 0 {
		master = hosts[0]
	}
	return []string{
		"KNIT_RANK=" + strconv.Itoa(rank),
		"KNIT_NNODES=" + strconv.Itoa(len(hosts)),
		"KNIT_HOSTS=" + strings.Join(hosts, ","),
		"KNIT_MASTER=" + master,
		"MLX_RANK=" + strconv.Itoa(rank),
		"MLX_HOSTFILE=" + hostfile,
	}
}

// Hostfile renders hosts in the JSON form MLX distributed reads: one entry per
// rank with the address it is reached at.
func Hostfile(hosts []string) []byte {
	type host struct {
		SSH string   `json:"ssh"`
		IPs []string `json:"ips"`
	}
	entries := make([]host, 0, len(hosts))
	for _, h := range hosts {
		entries = append(entries, host{SSH: h, IPs: []string{h}})
	}
	b, _ := json.Marshal(entries)
	return append(b, '\n')
}
