---
title: Commands
order: 1
---
# Commands

Seven commands. There is no configuration file and no eighth command.

## knit up [-d | --forever]

Start sharing this machine. Without a flag the agent runs in the foreground
until Ctrl-C. With `-d` it detaches, logs to `~/.knit/agent.log`, and runs
until `knit down` or logout. With `--forever` it is installed as a launchd
agent or systemd user unit that starts at login and restarts if it stops.
Already running: says so and exits 0. Creates the cluster key on first use if
there is none.

## knit down

Stop the agent however it was started: removes the `--forever` unit if one is
installed, else signals the detached agent. Not running: says so and exits 0.

## knit gauge [--json] [--peer HOST[:PORT]]...

List this machine and every reachable machine that holds the key, with live
capacity. Always browses fresh. `--json` emits the array described in
[Output formats](output.md). `knit ls` is an alias.

Columns: NAME, ADDR, OS/ARCH, CPUS, MEM (total), FREE (allocatable now), LOAD
(one-minute average), GPU, LINK (see [Cables, Ethernet, Wi-Fi](../guides/links.md)).

## knit run [flags] -- CMD [ARGS...]

Run a command on the machine with the most spare capacity, or where the flags
say. Streams stdin, stdout, and stderr; exits with the command's code.

| Flag | Effect |
| ---- | ------ |
| `--on NAME` | run on that machine; exit 126 if unreachable |
| `--mem GB` | only machines with at least GB allocatable now |
| `--arch ARCH` | only machines of that architecture (`arm64`, `amd64`) |
| `--dir` | send the working directory and run inside it |
| `--sync` | `--dir`, then copy created and changed files back |
| `--peer HOST[:PORT]` | also consider this machine; repeatable |

Everything after `--` is the command. Without `--dir` it runs in the agent
user's home directory on the target.

## knit each [--peer HOST[:PORT]]... -- CMD [ARGS...]

Run a command on every machine at once, this one included, each line prefixed
`[name] `. Exits with the highest exit code seen. Sets the rank environment on
every process; see [Environment variables](environment.md).

## knit key [--rotate]

Print this machine's cluster key as 64 hex characters, creating it if needed.
`--rotate` replaces it with a fresh key atomically, prints the new key, and
names the machines that were reachable under the old one and must now
`knit join` it. The local agent uses the new key from its next connection.

## knit join KEY

Install a key from another machine, replacing this machine's key. After this,
the two machines trust each other.

## knit --version, knit --help

Print the version, or the usage summary.
