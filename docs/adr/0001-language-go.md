# ADR-0001: Go as the implementation language
Status: Accepted
Date: 2026-09-01

## Context
knit is a single-binary CLI that must cross-compile to macOS and Linux on arm64
and amd64, move bytes between sockets and pipes with a few concurrent workers, do
HMAC and random-nonce auth, and be readable in an afternoon. The bottleneck in
production is a physical link (Thunderbolt/Ethernet/Wi-Fi), not CPU.

## Decision
Implement knit in Go.

## Alternatives considered
- **Rust** — marginally lower tail latency, no GC, stronger memory guarantees. Lost
  on build/read speed and ceremony: the hot path here does no allocation, so the
  GC never runs on it, erasing Rust's main advantage for this workload; and the
  "read in an afternoon" test favors Go's smaller surface.
- **C** — smallest binary, but hand-rolled concurrency and crypto plumbing is exactly
  the complexity knit exists to avoid.
- **Zig** — attractive and fast, but a smaller ecosystem for mDNS and a less
  familiar read for most contributors.
- **A scripting language (Python/Node)** — fails the single-static-binary and
  zero-runtime-install requirements outright.

## Consequences
- Static, CGo-free binaries cross-compiled from one machine; distribution is a file.
- Concurrency (goroutines + channels) matches the execution model directly.
- Stdlib covers crypto, net (`TCP_NODELAY`, `CloseWrite`), `os/exec`, binary/JSON —
  runtime dependency count is one (mDNS).
- We accept a GC, mitigated by a zero-allocation hot path (see
  [10-performance.md](../10-performance.md)); revisit only if profiling ever shows
  GC on the data path.
