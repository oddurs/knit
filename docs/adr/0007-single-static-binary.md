# ADR-0007: One static, CGo-free binary serving both roles
Status: Accepted
Date: 2026-09-01

## Context
Distribution and mental model are part of "seamless." Two binaries (agent + client),
or a binary that needs libc, means install friction, version skew, and platform
packaging pain.

## Decision
Ship exactly one binary. `connex up` runs the agent; every other subcommand is the
client; a machine is routinely both. Build with `CGO_ENABLED=0` so the result is a
fully static executable with no libc dependency; OS-specific sysinfo uses `/proc`
and `sysctl` via `os/exec`, never CGo.

## Alternatives considered
- **Separate agent and client binaries** — cleaner role separation on paper, but
  doubles the install/version surface and confuses the "one tool" story.
- **CGo for native sysinfo** (e.g. `sysctlbyname` bindings) — marginally cleaner
  than shelling out, but reintroduces libc, cross-compilation pain, and a
  non-static binary. Not worth it; the `os/exec` path is fast and already off the
  hot path.

## Consequences
- Distribution is "copy one file" or `brew install`; cross-compiles to four targets
  from one machine.
- No libc/glibc-version portability problems on Linux.
- Agent and client share the protocol, keys, and sysinfo code directly — no drift.
- Windows agent stays out of scope (needs job-object process control); a client-only
  Windows build remains conceivable without changing this decision.
