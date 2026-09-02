# Backlog

Measured future wins, deferred by choice rather than forgotten. They stay coded in
[`../registry.toml`](../registry.toml) with `milestone = "backlog"` so they are
never lost and their dependencies stay tracked.

- [`CX-XPORT-050`] — **connection reuse.** Upgrade the winning peer's `info`-probe
  connection into the following `run`, saving one dial + handshake. A real but small
  latency win ([performance](../../docs/10-performance.md)); v1 keeps
  one-op-per-connection to preserve the one-page protocol
  ([ADR-0005](../../docs/adr/0005-tcp-length-prefixed-framing.md)). Promote only if
  measurement shows the saved RTT matters against real command runtimes.

- [`CX-CORE-050`] — **Windows client.** The client subcommands are conceivable on
  Windows; the agent is explicitly out of scope because it needs job-object process
  control that isn't worth the effort yet ([technology](../../docs/07-technology.md)).

## Promotion

A backlog item earns a milestone when its dependencies are done and a concrete need
appears. Change its `milestone` and `status` in the registry, re-run
`roadmap.py check`, and re-render — the item keeps its ID and history.
