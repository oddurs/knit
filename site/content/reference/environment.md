---
title: Environment variables
order: 2
---
# Environment variables

All optional. Set them in your shell for the client side; the agent reads
`KNIT_HOME` only.

## Read by knit

| Variable | Effect |
| -------- | ------ |
| `KNIT_HOME` | directory for the key, pidfile, log, and caches; default `~/.knit` |
| `KNIT_PEERS` | comma-separated `host[:port]` list of machines to always consider |
| `KNIT_NO_CACHE` | if set, ignore the five-second peer cache and browse fresh every time |
| `KNIT_TIMEOUT_MS` | per-machine capacity probe budget in milliseconds; default 250 |
| `NO_COLOR` | if set, the `knit → name` line is printed without dimming |

Raise `KNIT_TIMEOUT_MS` when machines on a slow or distant link are missing
from `knit gauge`; a machine that cannot answer within the budget is left out of
that run rather than waited for.

## Set by knit each

Every process `knit each` launches, on this machine and on peers, gets:

| Variable | Value |
| -------- | ----- |
| `KNIT_RANK` | this machine's position in the launch; 0 is the machine `each` ran on |
| `KNIT_NNODES` | number of machines in the launch |
| `KNIT_HOSTS` | comma-separated addresses in rank order |
| `KNIT_MASTER` | address of rank 0 |
| `MLX_RANK` | same as `KNIT_RANK` |
| `MLX_HOSTFILE` | path of an MLX-format hostfile written for this launch |

Ranks after 0 follow the order of spare capacity. The hostfile is written under
`KNIT_HOME` for the duration of the command and removed afterwards.

Use `sh -c '...'` in the command when you need these expanded on the target;
your local shell would otherwise expand them before knit runs.
