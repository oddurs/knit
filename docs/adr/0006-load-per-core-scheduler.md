# ADR-0006: Load-per-core scheduling, local always a candidate
Status: Accepted
Date: 2026-09-01

## Context
connex must pick a machine for `run` with no config and no user thought in the
common case, while staying seamless: if the local machine is the best choice, the
command should just run locally with nothing printed.

## Decision
Score every candidate as `load1 / cpu_count` (lower is better) from a live `info`
probe; the local machine is always a candidate, scored the same way from local
sysinfo at no network cost. Lowest score wins. If local wins or there are no peers,
exec locally and print nothing. `--on NAME` pins a target and skips scoring;
`--mem N` / `--arch` (v0.3) are pre-scoring filters.

## Alternatives considered
- **Round-robin / random** — ignores real headroom; would offload onto a busy box.
- **Full resource modelling (CPU%, mem pressure, GPU, thermals)** — more accurate,
  far more complex and more to probe; premature. Load-per-core is a good, cheap
  proxy and the filters cover the cases that actually matter (memory for models).
- **A central scheduler / coordinator** — a server to run and keep alive; violates
  the no-server, zero-config posture.

## Consequences
- The decision is arithmetic over a few structs — sub-millisecond, no state.
- The invisible local fallback (print nothing when local wins) falls out naturally.
- "Dumb but honest" scheduling; refinements (memory/link/arch filters) are additive
  and already sequenced in the v0.3 roadmap.
- A brief `info` probe is required for every candidate, bounded by a 250 ms deadline
  so a slow peer never delays the decision.
