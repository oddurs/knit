// Command knit lets machines share compute with zero config. One binary plays
// both roles: `knit up` is the agent, everything else is the client.
// See docs/ and https://github.com/oddurs/knit.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/oddurs/knit/internal/agent"
	"github.com/oddurs/knit/internal/client"
	"github.com/oddurs/knit/internal/keys"
)

// version is injected at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `knit — weave your machines into one fabric. Zero config.

Usage:
  knit up [-d|--forever]   start sharing this machine
                           (-d: background; --forever: at every login, via
                           launchd/systemd, restarted if it stops)
  knit down                stop the agent, however it was started
  knit gauge [--json]      show machines and their capacity  (alias: ls)
  knit run [--on NAME] [--mem GB] [--arch ARCH] [--dir] [--sync] -- CMD [ARGS...]
                           run a command on the machine with most headroom
                           (--mem/--arch only consider machines that fit;
                           --dir sends the working dir; --sync mirrors changes back)
  knit each -- CMD [ARGS...]
                           run a command on every machine at once, with
                           KNIT_RANK/KNIT_NNODES/KNIT_HOSTS/KNIT_MASTER and
                           MLX_RANK/MLX_HOSTFILE set for multi-node launchers
  knit key [--rotate]      print this machine's cluster key
                           (--rotate: replace it and say who must re-join)
  knit join KEY            join the fabric that key belongs to
  knit proxy [--on NAME] [--port N]
                           tunnel this machine's network through a peer
                           (local SOCKS5, default port 1080)

  knit --version           print version

Add machines mDNS cannot see with --peer host[:port] (or KNIT_PEERS), e.g. over
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
		mode := agent.Foreground
		if hasFlag(args[1:], "-d") || hasFlag(args[1:], "--detach") {
			mode = agent.Detached
		}
		if hasFlag(args[1:], "--forever") {
			mode = agent.Forever
		}
		if err := agent.Up(mode); err != nil {
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
		if hasFlag(args[1:], "--rotate") {
			return client.Rotate()
		}
		k, err := keys.Print()
		if err != nil {
			fmt.Fprintln(os.Stderr, "knit:", err)
			return 1
		}
		fmt.Println(k)
		return 0
	case "proxy":
		rest, peers := extractPeers(args[1:])
		client.AddExplicitPeers(peers)
		return cmdProxy(rest)
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

// cmdRun parses `run [--on NAME] [--mem GB] [--arch ARCH] [--dir] [--sync] -- CMD...`.
func cmdRun(args []string) int {
	var p client.Placement
	i := 0
	for i < len(args) {
		switch {
		case args[i] == "--on" || args[i] == "--mem" || args[i] == "--arch":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "knit: %s needs a value\n", args[i])
				return client.ExitUsage
			}
			v := args[i+1]
			switch args[i] {
			case "--on":
				p.On = v
			case "--arch":
				p.Arch = v
			case "--mem":
				gb, err := strconv.ParseFloat(v, 64)
				if err != nil || gb <= 0 {
					fmt.Fprintf(os.Stderr, "knit: --mem needs gigabytes, e.g. --mem 48 (got %q)\n", v)
					return client.ExitUsage
				}
				p.MemGB = gb
			}
			i += 2
		case args[i] == "--dir":
			p.Dir = true
			i++
		case args[i] == "--sync":
			p.Sync = true
			i++
		case args[i] == "--":
			return client.Run(p, args[i+1:])
		default:
			fmt.Fprintf(os.Stderr, "knit: unexpected argument %q (did you forget `--` before the command?)\n", args[i])
			return client.ExitUsage
		}
	}
	fmt.Fprintln(os.Stderr, "knit: run needs `-- CMD`")
	return client.ExitUsage
}

// cmdProxy parses `proxy [--on NAME] [--port N]`.
func cmdProxy(args []string) int {
	on, port := "", 1080
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--on":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "knit: --on needs a machine name")
				return client.ExitUsage
			}
			on = args[i+1]
			i++
		case "--port":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "knit: --port needs a number")
				return client.ExitUsage
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 || n > 65535 {
				fmt.Fprintf(os.Stderr, "knit: --port needs a port number (got %q)\n", args[i+1])
				return client.ExitUsage
			}
			port = n
			i++
		default:
			fmt.Fprintf(os.Stderr, "knit: unexpected argument %q\n", args[i])
			return client.ExitUsage
		}
	}
	return client.Proxy(on, port)
}

// extractPeers pulls repeated `--peer host[:port]` flags from the arguments before
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
