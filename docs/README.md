# knit — documentation

**knit** is a single static binary that lets machines on the same desk, cable,
or LAN share compute with zero configuration. Run `knit up` and a machine
becomes discoverable capacity. Run `knit run -- <cmd>` anywhere and it executes
on whichever machine — including this one — has the most spare headroom, with
stdin/stdout/stderr and the exit code behaving byte-for-byte as if it ran locally.

## North star

The simplest possible version of this idea that still has **blazing-fast
internals** and a **fully seamless workflow**. Every decision is scored on three
axes, in this order:

1. **Seamless** — indistinguishable from running locally. No config, no accounts,
   no IPs, exit codes and pipes preserved exactly. Latency you cannot feel.
2. **Fast** — the transport is line-rate on a Thunderbolt cable; the client adds
   sub-millisecond overhead on the hot path; discovery never makes you wait twice.
3. **Small** — one binary, one real dependency, a wire protocol that fits on a
   page, a codebase you can read in an afternoon.

When two of these pull against each other, the order above breaks the tie.

## Reading order

| Doc | Contents |
| --- | --- |
| [01-vision.md](01-vision.md) | What knit is, the principles, why the cable is a solved problem |
| [02-architecture.md](02-architecture.md) | Components, package layout, concurrency model, data path, lifecycle |
| [03-protocol.md](03-protocol.md) | Wire protocol: handshake, auth, framing, error codes, versioning |
| [04-cli.md](04-cli.md) | Command surface, exit-code contract, output contracts, env vars |
| [05-ai-workloads.md](05-ai-workloads.md) | Distributed inference, memory-aware placement, network sharing |
| [07-technology.md](07-technology.md) | Language, dependency, and platform choices with rationale |
| [08-security-model.md](08-security-model.md) | Threat model, trust boundary, what v1 does and does not defend |
| [09-build-test-release.md](09-build-test-release.md) | Build, test matrix, benchmarks, smoke tests, release |
| [10-performance.md](10-performance.md) | Latency budget, hot paths, buffer strategy, benchmarking plan |
| [adr/](adr/) | Architecture Decision Records — one file per irreversible choice |

The staged plan lives in [`../roadmaps/`](../roadmaps/), not here: individual
milestone roadmaps plus a coded, machine-checkable work-item registry.

## Status

No code yet — this is the design baseline for a clean first implementation. The
v0.1 build target and every item after it are tracked in
[`../roadmaps/registry.toml`](../roadmaps/registry.toml); run
`python3 roadmaps/tools/roadmap.py check` to validate the plan and
`... render` to regenerate [`../roadmaps/STATUS.md`](../roadmaps/STATUS.md).
