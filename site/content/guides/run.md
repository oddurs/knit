---
title: Run a command elsewhere
order: 1
---
# Run a command elsewhere

```sh
knit run -- <command> [args...]
```

Everything after `--` is the command, passed through untouched. knit picks the
machine with the most spare capacity, including this one, and runs it there.

## What you see

When the work leaves this machine, knit prints one dim line to stderr:

```
knit → studio
```

When this machine wins, knit prints nothing and the command runs as if you had
typed it directly. Scripts can tell the difference by the line; humans mostly
stop noticing. Set `NO_COLOR` to get the line without the dimming.

## Pin a machine

```sh
knit run --on studio -- python quantize.py
```

`--on` takes the name shown in `knit gauge`. Naming this machine runs the
command locally. Naming a machine that is not reachable fails with exit code
126 rather than quietly running somewhere else.

## Pipes and stdin

stdin is streamed to the remote command as it arrives and closed when yours
closes, so pipes work at any size:

```sh
cat big.log | knit run -- zstd -19 > big.log.zst
knit run --on studio -- cat results.csv > local-copy.csv
```

Output is byte-for-byte what the command wrote; nothing is buffered, reflowed,
or re-encoded.

## Exit codes and signals

The command's exit code becomes knit's exit code, so `knit run -- false; echo
$?` prints 1 and `set -e` scripts behave. A command that dies from a signal
reports 128 plus the signal number, as a shell would.

Ctrl-C sends the real interrupt to the remote process, which can trap it and
exit on its own terms. If the connection is lost instead, the agent stops the
process itself, so nothing keeps running unattended on the other machine.

## Where the command runs

Without `--dir`, the command runs in the agent user's home directory on the
remote machine, with that machine's environment and tools. Relative paths refer
to files there, not here. To bring your working directory along, see
[Send files with the command](files.md).

The command must exist on the target. `knit run -- ffmpeg` on a machine without
ffmpeg fails the way it would locally, with exit code 127.
