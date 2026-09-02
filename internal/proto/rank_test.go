package proto

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestRankEnvAndHostfile(t *testing.T) {
	hosts := []string{"10.0.0.1", "10.0.0.2"}
	env := strings.Join(RankEnv(1, hosts, "/x/hosts.json"), "\n")
	for _, want := range []string{"KNIT_RANK=1", "KNIT_NNODES=2", "KNIT_HOSTS=10.0.0.1,10.0.0.2",
		"KNIT_MASTER=10.0.0.1", "MLX_RANK=1", "MLX_HOSTFILE=/x/hosts.json"} {
		if !strings.Contains(env, want) {
			t.Fatalf("missing %q in\n%s", want, env)
		}
	}
	var parsed []struct {
		SSH string   `json:"ssh"`
		IPs []string `json:"ips"`
	}
	if err := json.Unmarshal(Hostfile(hosts), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 || parsed[1].SSH != "10.0.0.2" || parsed[1].IPs[0] != "10.0.0.2" {
		t.Fatalf("got %+v", parsed)
	}
	if got := RankEnv(0, nil, "")[3]; got != "KNIT_MASTER=" {
		t.Fatalf("empty hosts: %q", got)
	}
}

func TestExitStatusShellConventions(t *testing.T) {
	if got := ExitStatus(nil); got != 0 {
		t.Fatalf("nil: %d", got)
	}
	if got := ExitStatus(exec.Command("definitely-not-a-program-xyz").Run()); got != 127 {
		t.Fatalf("missing program: %d, want 127", got)
	}
	if got := ExitStatus(exec.Command("sh", "-c", "exit 7").Run()); got != 7 {
		t.Fatalf("exit 7: %d", got)
	}
	if got := ExitStatus(exec.Command("sh", "-c", "kill -TERM $$").Run()); got != 143 {
		t.Fatalf("SIGTERM: %d, want 143", got)
	}
}
