---
title: Troubleshooting
order: 2
---
# Troubleshooting

## A machine is not listed

In order of likelihood:

1. **Its agent is not running.** On that machine, `knit up -d`. Check
   `~/.knit/agent.log` for the line saying it is up and on which port.
2. **It holds a different key.** `knit gauge` silently omits machines that
   refuse the key. On it, `knit join <key>` with the key from `knit key` here.
3. **Multicast does not reach it.** Different subnet, a router in between,
   a guest network, a VPN: name it with
   [`--peer`](../guides/peers.md) or `KNIT_PEERS`.
4. **It answered too slowly.** The probe budget is a quarter of a second. On a
   distant or sleepy link, `KNIT_TIMEOUT_MS=1000 knit gauge`.
5. **A firewall.** Allow TCP 5648 inbound on the agent machine. On macOS,
   accept the prompt to let `knit` receive connections.
6. **It runs an older knit.** Since v0.4 every connection is encrypted, and a
   v0.3 agent cannot answer a v0.4 client (or the reverse). Name it with
   `--peer` and `knit gauge` says `runs an older knit`; upgrade it.

`knit gauge` waits a full second for discovery every time, so it is not the
command to script in a tight loop; `knit run` uses a five-second cache and is
fast.

## unauthorized

```
knit: unauthorized on studio — run `knit key` there and `knit join <key>` here
```

The keys differ. Pick one machine's key and `knit join` it on the other. Keys
are per fabric, not per machine.

## The command runs, but in the wrong place

Without `--dir`, a remote command runs in the agent user's home directory on
the target. Relative paths mean files there. Use
[`--dir` or `--sync`](../guides/files.md) to bring your directory along, or pass
absolute paths that exist on the target.

## The command ran locally when I expected remote

This machine had the most headroom, so it won. That is the intended default.
Use `--on` to insist, or `--mem` to state what the job needs.

## Exit code 126 or 127

126: no machine could take the job: `--on` named something unreachable, or
nothing passed `--mem`/`--arch`. The message names the constraint. 127 with
`unauthorized`: the keys differ. 127 with `executable file not found`: the
command does not exist on the target machine.

## Something is still running on the other machine

It should not be: when a client disconnects for any reason, the agent sends the
process an interrupt and, two seconds later, kills it along with everything it
started. If you see an orphan, the agent's log will show the run; please report
it with the command and both versions.

## A stale agent

If `knit down` reports no agent but `knit gauge` still shows this machine, an
agent process is running that knit did not start: a `--forever` unit from a
different `KNIT_HOME`, or a leftover process without a pidfile. Check
`launchctl print gui/$(id -u)/io.knit.agent` on macOS or `systemctl --user
status knit` on Linux; otherwise `pgrep -f "knit up"` and `kill <pid>`.

## Two agents, one name

Two machines with the same short hostname are indistinguishable to `--on`.
Rename one; knit uses the hostname as the machine's name.
