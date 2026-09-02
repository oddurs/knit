---
title: Files
order: 5
---
# Files

Everything knit keeps lives in one directory, `~/.knit` (or `KNIT_HOME`),
created with mode 0700 on first use.

| File | What |
| ---- | ---- |
| `key` | the cluster key, 64 hex characters, mode 0600. Back this up or rotate with `knit key --rotate`; nothing else identifies the fabric |
| `agent.pid` | pid of the background agent, written by `knit up -d`, removed by `knit down` |
| `agent.log` | the background agent's log: the port it bound, each command it ran |
| `peers.json` | the five-second cache of discovered addresses; never holds capacity, safe to delete |
| `hostfile-*.json` | a per-launch MLX hostfile while a `knit each` command runs; removed afterwards |

There is no configuration file, and knit reads nothing outside this directory.

To start over on a machine, stop the agent and delete the directory:

```sh
knit down && rm -r ~/.knit
```

The next `knit up` or `knit key` makes a fresh key, which means re-pairing
with `knit join`.
