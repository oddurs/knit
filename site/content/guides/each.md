---
title: Run on every machine
order: 3
---
# Run on every machine

```sh
knit each -- <command> [args...]
```

The command starts on every machine in the fabric at once, this one included,
and their output is interleaved as it arrives.

## Prefixed lines

Every line is prefixed with the machine it came from, on stdout and stderr
alike, and lines are never split mid-way even when machines talk over each
other:

```
$ knit each -- uname -sr
[here] Darwin 25.5.0
[studio] Darwin 25.5.0
[box] Linux 6.12.0
knit each: here=0 studio=0 box=0
```

The last line, on stderr and dimmed, is the per-machine exit summary.

## Exit code

`knit each` exits 0 only if every machine's command exited 0. Otherwise it exits
with the highest code seen, so a failure anywhere fails the whole command.

## No stdin

`each` gives every command an empty stdin. A command that waits for input gets
end-of-file immediately.

## Rank and hosts

Every process `each` launches gets the environment a multi-node launcher needs:
its rank, the number of machines, the address list in rank order, and the
address of rank 0. This machine is rank 0; the others follow in order of spare
capacity. See [AI workloads](ai-workloads.md) for what that enables and
[Environment variables](../reference/environment.md) for the names.

## Typical uses

```sh
knit each -- knit --version            # everyone on the same build?
knit each -- df -h /                    # disk on every machine
knit each -- sh -c 'brew upgrade knit'  # one shell per machine
```

Use `sh -c '...'` when the command needs a shell: variables, pipes, or
redirections are otherwise interpreted by your local shell before knit sees
them.
