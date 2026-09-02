# knit — plan status

> Generated from `registry.toml` by `roadmaps/tools/roadmap.py render`.
> Do not hand-edit — change the registry and re-render.

**36 items** across **4 milestones** plus backlog · **32 done**

## Rollup

| Status | Count |   | Priority | Count |
| ------ | ----: | - | -------- | ----: |
| ready | 1 |   | p0 | 15 |
| done | 32 |   | p1 | 14 |
| deferred | 3 |   | p2 | 7 |

## m1 · v0.1 — the fabric

*Two machines, one cable or LAN: run / each / ls working end to end, fast and seamless.*

```
███████████████████████░  19/20
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 0 | `KN-AUTH-001` | Keyfile load/create and HMAC-nonce sign/verify | done | p0 | S | — |
| 2 | `KN-CLI-001` | key and join commands | done | p0 | S | KN-AUTH-001 |
| 3 | `KN-DISC-001` | mDNS register (agent) and browse (client) on all interfaces | done | p0 | M | — |
| 4 | `KN-DISC-002` | Short client-side peer cache (~5s TTL) | done | p1 | S | KN-DISC-001 |
| 5 | `KN-PROTO-001` | Wire types, framing, version token, error codes | done | p0 | M | — |
| 6 | `KN-SYS-001` | System info: cores, total memory, load1 (darwin+linux) | done | p0 | S | — |
| 7 | `KN-AGENT-001` | Agent: listener, auth gate, op dispatch, up/up -d/down | done | p0 | L | KN-PROTO-001, KN-AUTH-001, KN-SYS-001, KN-DISC-001 |
| 8 | `KN-EXEC-001` | Executor: spawn, stream stdio, relay exit code | done | p0 | L | KN-AGENT-001, KN-PROTO-001 |
| 9 | `KN-EXEC-010` | SIGINT forwarding: Ctrl-C reaches the remote process, no orphan | done | p1 | M | KN-EXEC-001 |
| 13 | `KN-SCHED-001` | Load-per-core scheduler with local candidate and --on | done | p0 | S | KN-SYS-001 |
| 16 | `KN-XPORT-001` | Dial + handshake helper with socket tuning | done | p0 | M | KN-PROTO-001, KN-AUTH-001 |
| 18 | `KN-CLIENT-001` | knit gauge: parallel info probes, table + --json | done | p0 | M | KN-XPORT-001, KN-DISC-002, KN-SYS-001 |
| 20 | `KN-CLIENT-002` | knit run: schedule, local fallback, stream, exit relay | done | p0 | L | KN-SCHED-001, KN-XPORT-001, KN-EXEC-001, KN-CLIENT-001 |
| 21 | `KN-CLIENT-003` | knit each: fan-out with per-line prefix and aggregate exit | done | p1 | M | KN-CLIENT-002 |
| 24 | `KN-CORE-001` | main(): command dispatch, usage, exit-code contract | done | p0 | M | KN-CLIENT-001, KN-CLIENT-002, KN-CLIENT-003, KN-CLI-001 |
| 27 | `KN-DOC-001` | Repo-root README, SECURITY.md, quickstart | done | p1 | S | KN-CORE-001 |
| 29 | `KN-OPS-001` | Static, CGo-free build with version stamping | done | p0 | S | KN-CORE-001 |
| 32 | `KN-TEST-001` | Test harness: unit, protocol, loopback e2e, race, fuzz | done | p0 | M | KN-CORE-001 |
| 33 | `KN-TEST-002` | CI: fmt, vet, staticcheck, race, cross-build, roadmap check | done | p1 | S | KN-TEST-001 |
| 34 | `KN-TEST-003` | Two-machine manual matrix: Thunderbolt bridge + Wi-Fi | ready | p1 | S | KN-CORE-001 |

## m2 · v0.2 — trustworthy under real use

*Signals, explicit peers, working-dir sync, and a real install path.*

```
████████████████████████  6/6
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 10 | `KN-EXEC-020` | Robust signal forwarding (SIGINT/SIGTERM) groundwork for KNIT2 | done | p1 | M | KN-EXEC-010 |
| 11 | `KN-EXEC-030` | run --dir / --sync: rsync-style working-directory sync | done | p1 | L | KN-EXEC-001 |
| 22 | `KN-CLIENT-020` | ls marks the interface/link a peer was found on | done | p2 | S | KN-CLIENT-001, KN-DISC-001 |
| 26 | `KN-DISC-020` | --peer host:port and KNIT_PEERS (works over Tailscale) | done | p1 | S | KN-XPORT-001 |
| 30 | `KN-OPS-020` | goreleaser release pipeline | done | p1 | S | KN-OPS-001 |
| 31 | `KN-OPS-021` | Homebrew tap: brew install oddurs/tap/knit | done | p1 | S | KN-OPS-020 |

## m3 · v0.3 — AI-native scheduling

*Free-memory and accelerator reporting, memory-aware placement, MLX/torchrun sugar, link awareness.*

```
████████████████████████  4/4
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 14 | `KN-SYS-030` | info grows mem_free_gb, gpu, accel | done | p1 | M | KN-SYS-001 |
| 15 | `KN-SCHED-030` | run --mem N and --arch placement filters | done | p1 | S | KN-SYS-030, KN-SCHED-001 |
| 19 | `KN-AI-030` | Ranked multi-node launch: each exports rank, hosts, and an MLX hostfile | done | p2 | M | KN-SYS-030, KN-CLIENT-001 |
| 23 | `KN-CLIENT-030` | Link-speed awareness: prefer wired address, show link in ls | done | p2 | M | KN-CLIENT-020, KN-DISC-001 |

## m4 · v0.4 — encrypted, persistent, shareable

*TLS with pinned certs, reboot-persistent agents, and network sharing via SOCKS.*

```
██████████████████░░░░░░  3/4
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 1 | `KN-AUTH-041` | Key rotation UX: knit key --rotate | done | p2 | S | KN-AUTH-001 |
| 12 | `KN-OPS-040` | launchd/systemd install: knit up --forever | done | p1 | M | KN-AGENT-001 |
| 17 | `KN-AUTH-040` | TLS 1.3 with a key-bound handshake | done | p0 | L | KN-XPORT-001, KN-AUTH-001 |
| 28 | `KN-NET-040` | knit proxy: share a peer's network via local SOCKS5 | deferred | p2 | L | KN-AUTH-040 |

## backlog · — — backlog

*Measured future wins deferred by choice, kept coded so they are never lost.*

```
░░░░░░░░░░░░░░░░░░░░░░░░  0/2
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 25 | `KN-CORE-050` | Windows client build (agent stays out of scope) | deferred | p2 | M | KN-CORE-001 |
| 35 | `KN-XPORT-050` | Connection reuse: upgrade the info probe into the following run | deferred | p2 | M | KN-XPORT-001, KN-CLIENT-002 |

## Blocked / deferred

- `KN-NET-040` **deferred** — knit proxy: share a peer's network via local SOCKS5 · Deferred 2026-09-02 with TLS in place: it would be the eighth command (docs/04-cli.md: seven by design), and its use cases are already served — a Tailscale exit node for the laptop-on-hotel-Wi-Fi case, macOS Internet Sharing for a Thunderbolt-only machine. Revisit only if real usage asks for it.
- `KN-XPORT-050` **deferred** — Connection reuse: upgrade the info probe into the following run · Deferred: real but small win; v1 keeps one-op-per-connection for the one-page protocol.
- `KN-CORE-050` **deferred** — Windows client build (agent stays out of scope) · Deferred: client is conceivable; agent needs job-object work not worth it yet.

## Suggested build order (dependency topological sort)

The number in each milestone table is this global order. m1 critical path first:

```
 0. KN-AUTH-001    p0 S  Keyfile load/create and HMAC-nonce sign/verify
 2. KN-CLI-001     p0 S  key and join commands
 3. KN-DISC-001    p0 M  mDNS register (agent) and browse (client) on all interfaces
 4. KN-DISC-002    p1 S  Short client-side peer cache (~5s TTL)
 5. KN-PROTO-001   p0 M  Wire types, framing, version token, error codes
 6. KN-SYS-001     p0 S  System info: cores, total memory, load1 (darwin+linux)
 7. KN-AGENT-001   p0 L  Agent: listener, auth gate, op dispatch, up/up -d/down
 8. KN-EXEC-001    p0 L  Executor: spawn, stream stdio, relay exit code
 9. KN-EXEC-010    p1 M  SIGINT forwarding: Ctrl-C reaches the remote process, no orphan
13. KN-SCHED-001   p0 S  Load-per-core scheduler with local candidate and --on
16. KN-XPORT-001   p0 M  Dial + handshake helper with socket tuning
18. KN-CLIENT-001  p0 M  knit gauge: parallel info probes, table + --json
20. KN-CLIENT-002  p0 L  knit run: schedule, local fallback, stream, exit relay
21. KN-CLIENT-003  p1 M  knit each: fan-out with per-line prefix and aggregate exit
24. KN-CORE-001    p0 M  main(): command dispatch, usage, exit-code contract
27. KN-DOC-001     p1 S  Repo-root README, SECURITY.md, quickstart
29. KN-OPS-001     p0 S  Static, CGo-free build with version stamping
32. KN-TEST-001    p0 M  Test harness: unit, protocol, loopback e2e, race, fuzz
33. KN-TEST-002    p1 S  CI: fmt, vet, staticcheck, race, cross-build, roadmap check
34. KN-TEST-003    p1 S  Two-machine manual matrix: Thunderbolt bridge + Wi-Fi
```

