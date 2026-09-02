# ADR-0003: mDNS discovery + short client-side peer cache
Status: Accepted
Date: 2026-09-01

## Context
Zero config is the headline feature: no IPs, no hostfiles. Peers must find each
other on any link — Wi-Fi, Ethernet, and (critically) the Thunderbolt/USB4 bridge
interfaces macOS and Linux create automatically when a cable is plugged in. But a
cold mDNS browse costs ~300–600 ms, and paying that on every command would make
connex feel laggy — the opposite of seamless.

## Decision
Discover peers via multicast-DNS (`_connex._tcp`), browsing on all interfaces, and
cache the resulting `name → addr:port` mapping at `~/.connex/peers.json` with a
short TTL (~5 s). `connex ls` always browses fresh and refreshes the cache. Live
capacity (load, free memory) is **never** cached — it is always obtained by a fresh
`info` probe. `--peer host:port` / `CONNEX_PEERS` bypasses mDNS for filtered
networks (e.g. Tailscale). Dependency: `github.com/grandcat/zeroconf`.

## Alternatives considered
- **Fresh browse every command, no cache** (the original v0.1 sketch) — simplest,
  but the repeated ~0.5 s stall fails the seamless bar for build loops and
  interactive sessions.
- **A client-side discovery daemon** — 0 ms always, but adds a second long-lived
  process and stale-state bugs; too much machinery for the win.
- **Capacity in mDNS TXT records** — would let `ls` skip probes, but TXT goes stale
  instantly for load/free-mem and would mis-schedule. Rejected.
- **A custom UDP broadcast protocol** — removes the dependency but re-implements a
  solved problem worse; loses Bonjour interop.

## Consequences
- Back-to-back commands are instant; the first pays the browse.
- Placement stays correct because load is always live; a stale cached address just
  fails its probe and is dropped.
- One runtime dependency enters the tree. Fallback if it ever rots: vendor a
  minimal browser or switch to `hashicorp/mdns` — the `discovery` package is the
  only code that would change.
- `CONNEX_NO_CACHE` exists for anyone who wants the old always-fresh behavior.
