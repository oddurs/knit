// Command connex lets machines share compute with zero config. One binary plays
// both roles: `connex up` is the agent, everything else is the client.
// See docs/ and https://github.com/oddurs/connex.
package main

import (
	"fmt"
	"os"

	"github.com/oddurs/connex/internal/agent"
	"github.com/oddurs/connex/internal/client"
	"github.com/oddurs/connex/internal/keys"
)

// version is injected at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `connex — share compute across machines, zero config.

Usage:
  connex up [-d]              start sharing this machine's compute (-d: background)
  connex down                stop the background agent
  connex ls [--json]         list machines and live capacity
  connex run [--on NAME] -- CMD [ARGS...]
                             run a command wherever there is most headroom
  connex each -- CMD [ARGS...]
                             run a command on every machine at once
  connex key                 print this machine's cluster key
  connex join KEY            trust the cluster that key belongs to

  connex --version           print version
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
		fmt.Println("connex", version)
		return 0
	case "up":
		detach := len(args) > 1 && (args[1] == "-d" || args[1] == "--detach")
		if err := agent.Up(detach); err != nil {
			fmt.Fprintln(os.Stderr, "connex:", err)
			return 1
		}
		return 0
	case "down":
		if err := agent.Down(); err != nil {
			fmt.Fprintln(os.Stderr, "connex:", err)
			return 1
		}
		return 0
	case "ls":
		return client.LS(hasFlag(args[1:], "--json"))
	case "run":
		return cmdRun(args[1:])
	case "each":
		cmd, ok := afterDashDash(args[1:])
		if !ok {
			fmt.Fprintln(os.Stderr, "connex: each needs `-- CMD`")
			return client.ExitUsage
		}
		return client.Each(cmd)
	case "key":
		k, err := keys.Print()
		if err != nil {
			fmt.Fprintln(os.Stderr, "connex:", err)
			return 1
		}
		fmt.Println(k)
		return 0
	case "join":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "connex: join needs a key (get it with `connex key` on a trusted machine)")
			return client.ExitUsage
		}
		if err := keys.Save(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "connex:", err)
			return client.ExitUsage
		}
		fmt.Println("connex: cluster key installed — this machine now trusts that cluster")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "connex: unknown command %q\n\n%s", args[0], usage)
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
				fmt.Fprintln(os.Stderr, "connex: --on needs a machine name")
				return client.ExitUsage
			}
			on = args[i+1]
			i += 2
		case args[i] == "--":
			return client.Run(on, args[i+1:])
		default:
			fmt.Fprintf(os.Stderr, "connex: unexpected argument %q (did you forget `--` before the command?)\n", args[i])
			return client.ExitUsage
		}
	}
	fmt.Fprintln(os.Stderr, "connex: run needs `-- CMD`")
	return client.ExitUsage
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
