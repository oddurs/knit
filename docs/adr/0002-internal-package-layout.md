# ADR-0002: Small internal packages, not one flat `main`
Status: Accepted — supersedes the vision's "flat Go package" wording
Date: 2026-09-01

## Context
The original vision said "flat Go package" as shorthand for "small and readable."
As knit grows to agent + client + scheduler + discovery + protocol + auth +
sysinfo, a single flat package means shared mutable globals, no compiler-enforced
boundaries, and test files that can reach into anything. That works against
readability at the real size, not for it.

## Decision
Use a thin layer of single-purpose `internal/` packages with strictly downward
dependencies, plus `cmd/knit` for `main`. Each package targets a few hundred
lines; the whole tree still fits an afternoon read. Layout is listed in
[02-architecture.md](../02-architecture.md#package-layout).

## Alternatives considered
- **Truly flat single package** — the literal original wording. Lost: no enforced
  boundaries, global state, harder isolated tests; "small" is better served by
  small *packages* than one big file.
- **Deep hexagonal / DDD layering** — massively over-built for a tool this size;
  violates simplicity outright.

## Consequences
- Compiler enforces the dependency direction; each unit tests in isolation.
- A hard rule keeps it honest: if the tree stops fitting an afternoon's read, the
  feature is too big for knit.
- The vision doc's "single binary you can read in an afternoon" stands; only the
  "flat package" phrasing is retired, here, on purpose.
