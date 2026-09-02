// Command knit lets machines share compute with zero config. One binary plays
// both roles: `knit up` is the agent, everything else is the client.
// See docs/ and https://github.com/oddurs/knit.
package main

import (
	"fmt"
	"os"

	"github.com/oddurs/knit/internal/agent"
	"github.com/oddurs/knit/internal/client"
	"github.com/oddurs/knit/internal/keys"
)

// version is injected at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `knit — weave your machines into one fabric. Zero config.

Usage:
  knit up [-d]              start sharing this machine (-d: background)
  knit down                stop the background agent
  knit gauge [--json]      show machines and their capacity  (alias: ls)
  knit run [--on NAME] -- CMD [ARGS...]
                           run a command on the machine with most headroom
  knit each -- CMD [ARGS...]
                           run a command on every machine at once
  knit key                 print this machine's cluster key
  knit join KEY            join the fabric that key belongs to

  knit --version           print version

Add machines mDNS cannot see with --peer host:port (or KNIT_PEERS), e.g. over
Tailscale. Each machine is a loop; together they are the fabric.
`

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return client.ExitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(usage)
		return 0
	case "--version", "version":
		fmt.Println("knit", version)
		return 0
	case "up":
		detach := len(args) > 1 && (args[1] == "-d" || args[1] == "--detach")
		if err := agent.Up(detach); err != nil {
			fmt.Fprintln(os.Stderr, "knit:", err)
			return 1
		}
		return 0
	case "down":
		if err := agent.Down(); err != nil {
			fmt.Fprintln(os.Stderr, "knit:", err)
			return 1
		}
		return 0
	case "gauge", "ls": // ls kept as an alias for muscle memory
		rest, peers := extractPeers(args[1:])
		client.AddExplicitPeers(peers)
		return client.Gauge(hasFlag(rest, "--json"))
	case "run":
		rest, peers := extractPeers(args[1:])
		client.AddExplicitPeers(peers)
		return cmdRun(rest)
	case "each":
		rest, peers := extractPeers(args[1:])
		client.AddExplicitPeers(peers)
		cmd, ok := afterDashDash(rest)
		if !ok {
			fmt.Fprintln(os.Stderr, "knit: each needs `-- CMD`")
			return client.ExitUsage
		}
		return client.Each(cmd)
	case "key":
		k, err := keys.Print()
		if err != nil {
			fmt.Fprintln(os.Stderr, "knit:", err)
			return 1
		}
		fmt.Println(k)
		return 0
	case "join":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "knit: join needs a key (get it with `knit key` on a trusted machine)")
			return client.ExitUsage
		}
		if err := keys.Save(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "knit:", err)
			return client.ExitUsage
		}
		fmt.Println("knit: cluster key installed — this machine now trusts that cluster")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "knit: unknown command %q\n\n%s", args[0], usage)
		return client.ExitUsage
	}
}

// cmdRun parses `run [--on NAME] -- CMD...`.
func cmdRun(args []string) int {
	on := ""
	i := 0
	for i < len(args) {
		switch {
		case args[i] == "--on":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "knit: --on needs a machine name")
				return client.ExitUsage
			}
			on = args[i+1]
			i += 2
		case args[i] == "--":
			return client.Run(on, args[i+1:])
		default:
			fmt.Fprintf(os.Stderr, "knit: unexpected argument %q (did you forget `--` before the command?)\n", args[i])
			return client.ExitUsage
		}
	}
	fmt.Fprintln(os.Stderr, "knit: run needs `-- CMD`")
	return client.ExitUsage
}

// extractPeers pulls repeated `--peer host:port` flags from the arguments before
// a `--` separator, returning the remaining args and the collected peers.
func extractPeers(args []string) (rest []string, peers []string) {
	i := 0
	for i < len(args) {
		if args[i] == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		if args[i] == "--peer" {
			if i+1 < len(args) {
				peers = append(peers, args[i+1])
				i += 2
				continue
			}
			i++
			continue
		}
		rest = append(rest, args[i])
		i++
	}
	return rest, peers
}

func afterDashDash(args []string) ([]string, bool) {
	for i, a := range args {
		if a == "--" {
			return args[i+1:], true
		}
	}
	return nil, false
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
