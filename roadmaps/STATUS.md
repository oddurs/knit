# connex — plan status

> Generated from `registry.toml` by `roadmaps/tools/roadmap.py render`.
> Do not hand-edit — change the registry and re-render.

**36 items** across **4 milestones** plus backlog · **19 done**

## Rollup

| Status | Count |   | Priority | Count |
| ------ | ----: | - | -------- | ----: |
| ready | 1 |   | p0 | 15 |
| done | 19 |   | p1 | 14 |
| planned | 13 |   | p2 | 7 |
| blocked | 1 |   |  |  |
| deferred | 2 |   |  |  |

## m1 · v0.1 — the fabric

*Two machines, one cable or LAN: run / each / ls working end to end, fast and seamless.*

```
███████████████████████░  19/20
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 0 | `CX-AUTH-001` | Keyfile load/create and HMAC-nonce sign/verify | done | p0 | S | — |
| 2 | `CX-CLI-001` | key and join commands | done | p0 | S | CX-AUTH-001 |
| 3 | `CX-DISC-001` | mDNS register (agent) and browse (client) on all interfaces | done | p0 | M | — |
| 4 | `CX-DISC-002` | Short client-side peer cache (~5s TTL) | done | p1 | S | CX-DISC-001 |
| 5 | `CX-PROTO-001` | Wire types, framing, version token, error codes | done | p0 | M | — |
| 6 | `CX-SYS-001` | System info: cores, total memory, load1 (darwin+linux) | done | p0 | S | — |
| 7 | `CX-AGENT-001` | Agent: listener, auth gate, op dispatch, up/up -d/down | done | p0 | L | CX-PROTO-001, CX-AUTH-001, CX-SYS-001, CX-DISC-001 |
| 8 | `CX-EXEC-001` | Executor: spawn, stream stdio, relay exit code | done | p0 | L | CX-AGENT-001, CX-PROTO-001 |
| 9 | `CX-EXEC-010` | SIGINT forwarding: Ctrl-C reaches the remote process, no orphan | done | p1 | M | CX-EXEC-001 |
| 13 | `CX-SCHED-001` | Load-per-core scheduler with local candidate and --on | done | p0 | S | CX-SYS-001 |
| 16 | `CX-XPORT-001` | Dial + handshake helper with socket tuning | done | p0 | M | CX-PROTO-001, CX-AUTH-001 |
| 18 | `CX-CLIENT-001` | connex ls: parallel info probes, table + --json | done | p0 | M | CX-XPORT-001, CX-DISC-002, CX-SYS-001 |
| 20 | `CX-CLIENT-002` | connex run: schedule, local fallback, stream, exit relay | done | p0 | L | CX-SCHED-001, CX-XPORT-001, CX-EXEC-001, CX-CLIENT-001 |
| 21 | `CX-CLIENT-003` | connex each: fan-out with per-line prefix and aggregate exit | done | p1 | M | CX-CLIENT-002 |
| 24 | `CX-CORE-001` | main(): command dispatch, usage, exit-code contract | done | p0 | M | CX-CLIENT-001, CX-CLIENT-002, CX-CLIENT-003, CX-CLI-001 |
| 27 | `CX-DOC-001` | Repo-root README, SECURITY.md, quickstart | done | p1 | S | CX-CORE-001 |
| 29 | `CX-OPS-001` | Static, CGo-free build with version stamping | done | p0 | S | CX-CORE-001 |
| 32 | `CX-TEST-001` | Test harness: unit, protocol, loopback e2e, race, fuzz | done | p0 | M | CX-CORE-001 |
| 33 | `CX-TEST-002` | CI: fmt, vet, staticcheck, race, cross-build, roadmap check | done | p1 | S | CX-TEST-001 |
| 34 | `CX-TEST-003` | Two-machine manual matrix: Thunderbolt bridge + Wi-Fi | ready | p1 | S | CX-CORE-001 |

## m2 · v0.2 — trustworthy under real use

*Signals, explicit peers, working-dir sync, and a real install path.*

```
░░░░░░░░░░░░░░░░░░░░░░░░  0/6
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 10 | `CX-EXEC-020` | Robust signal forwarding (SIGINT/SIGTERM) groundwork for CONNEX2 | planned | p1 | M | CX-EXEC-010 |
| 11 | `CX-EXEC-030` | run --dir / --sync: rsync-style working-directory sync | planned | p1 | L | CX-EXEC-001 |
| 22 | `CX-CLIENT-020` | ls marks the interface/link a peer was found on | planned | p2 | S | CX-CLIENT-001, CX-DISC-001 |
| 26 | `CX-DISC-020` | --peer host:port and CONNEX_PEERS (works over Tailscale) | planned | p1 | S | CX-XPORT-001 |
| 30 | `CX-OPS-020` | goreleaser release pipeline | planned | p1 | S | CX-OPS-001 |
| 31 | `CX-OPS-021` | Homebrew tap: brew install connex/tap/connex | planned | p1 | S | CX-OPS-020 |

## m3 · v0.3 — AI-native scheduling

*Free-memory and accelerator reporting, memory-aware placement, MLX/torchrun sugar, link awareness.*

```
░░░░░░░░░░░░░░░░░░░░░░░░  0/4
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 14 | `CX-SYS-030` | info grows mem_free_gb, gpu, accel | planned | p1 | M | CX-SYS-001 |
| 15 | `CX-SCHED-030` | run --mem N and --arch placement filters | planned | p1 | S | CX-SYS-030, CX-SCHED-001 |
| 19 | `CX-AI-030` | connex hostfile and connex mpirun sugar | planned | p2 | M | CX-SYS-030, CX-CLIENT-001 |
| 23 | `CX-CLIENT-030` | Link-speed awareness: prefer wired address, show link in ls | planned | p2 | M | CX-CLIENT-020, CX-DISC-001 |

## m4 · v0.4 — encrypted, persistent, shareable

*TLS with pinned certs, reboot-persistent agents, and network sharing via SOCKS.*

```
░░░░░░░░░░░░░░░░░░░░░░░░  0/4
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 1 | `CX-AUTH-041` | Key rotation UX | planned | p2 | S | CX-AUTH-001 |
| 12 | `CX-OPS-040` | launchd/systemd install: connex up --forever | planned | p1 | M | CX-AGENT-001 |
| 17 | `CX-AUTH-040` | TLS 1.3 with pinned self-signed certs | planned | p0 | L | CX-XPORT-001, CX-AUTH-001 |
| 28 | `CX-NET-040` | connex proxy: share a peer's network via local SOCKS5 | blocked | p2 | L | CX-AUTH-040 |

## backlog · — — backlog

*Measured future wins deferred by choice, kept coded so they are never lost.*

```
░░░░░░░░░░░░░░░░░░░░░░░░  0/2
```

| # | ID | Item | Status | Pri | Sz | Depends |
| -: | -- | ---- | ------ | --- | -- | ------- |
| 25 | `CX-CORE-050` | Windows client build (agent stays out of scope) | deferred | p2 | M | CX-CORE-001 |
| 35 | `CX-XPORT-050` | Connection reuse: upgrade the info probe into the following run | deferred | p2 | M | CX-XPORT-001, CX-CLIENT-002 |

## Blocked / deferred

- `CX-NET-040` **blocked** — connex proxy: share a peer's network via local SOCKS5 · Blocked on CX-AUTH-040 by policy: proxied traffic is too sensitive for a plaintext link.
- `CX-XPORT-050` **deferred** — Connection reuse: upgrade the info probe into the following run · Deferred: real but small win; v1 keeps one-op-per-connection for the one-page protocol.
- `CX-CORE-050` **deferred** — Windows client build (agent stays out of scope) · Deferred: client is conceivable; agent needs job-object work not worth it yet.

## Suggested build order (dependency topological sort)

The number in each milestone table is this global order. m1 critical path first:

```
 0. CX-AUTH-001    p0 S  Keyfile load/create and HMAC-nonce sign/verify
 2. CX-CLI-001     p0 S  key and join commands
 3. CX-DISC-001    p0 M  mDNS register (agent) and browse (client) on all interfaces
 4. CX-DISC-002    p1 S  Short client-side peer cache (~5s TTL)
 5. CX-PROTO-001   p0 M  Wire types, framing, version token, error codes
 6. CX-SYS-001     p0 S  System info: cores, total memory, load1 (darwin+linux)
 7. CX-AGENT-001   p0 L  Agent: listener, auth gate, op dispatch, up/up -d/down
 8. CX-EXEC-001    p0 L  Executor: spawn, stream stdio, relay exit code
 9. CX-EXEC-010    p1 M  SIGINT forwarding: Ctrl-C reaches the remote process, no orphan
13. CX-SCHED-001   p0 S  Load-per-core scheduler with local candidate and --on
16. CX-XPORT-001   p0 M  Dial + handshake helper with socket tuning
18. CX-CLIENT-001  p0 M  connex ls: parallel info probes, table + --json
20. CX-CLIENT-002  p0 L  connex run: schedule, local fallback, stream, exit relay
21. CX-CLIENT-003  p1 M  connex each: fan-out with per-line prefix and aggregate exit
24. CX-CORE-001    p0 M  main(): command dispatch, usage, exit-code contract
27. CX-DOC-001     p1 S  Repo-root README, SECURITY.md, quickstart
29. CX-OPS-001     p0 S  Static, CGo-free build with version stamping
32. CX-TEST-001    p0 M  Test harness: unit, protocol, loopback e2e, race, fuzz
33. CX-TEST-002    p1 S  CI: fmt, vet, staticcheck, race, cross-build, roadmap check
34. CX-TEST-003    p1 S  Two-machine manual matrix: Thunderbolt bridge + Wi-Fi
```

