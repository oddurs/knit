---
title: Exit codes
order: 3
---
# Exit codes

The command's own exit code passes through untouched. knit's own failures use
codes a command rarely does, so scripts can tell them apart.

| Code | Who | Meaning |
| ---- | --- | ------- |
| 0–125 | the command | its own exit status, verbatim, local or remote |
| 128+n | the command | ended by signal n, as a shell reports it (130 after Ctrl-C) |
| 2 | knit | usage error: a bad flag, a missing `--`, `--mem` without a number |
| 124 | knit | the connection dropped before the command's exit code arrived |
| 126 | knit | no machine could take the job: `--on` unreachable, or nothing passed `--mem`/`--arch` |
| 127 | knit | unauthorized: the target does not hold this key |

`knit each` exits 0 only if every machine exited 0, else with the highest code
observed.

A command that does not exist on the target exits 127 too, exactly as a local
shell would; the message on stderr tells the two apart.
