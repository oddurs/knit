# Architecture

## Components

```
┌─────────────────────────── machine A (client role) ─────────────────────────┐
│  connex CLI ──┬── discovery   (mDNS browse on all interfaces, + short cache) │
│               ├── prober      (parallel `info` dials, 250 ms budget)         │
│               ├── scheduler   (pick lowest load-per-core; local is default)  │
│               └── client      (TCP, TCP_NODELAY, streamed stdio, exit relay) │
└───────────────────────────────────┬──────────────────────────────────────────┘
                                    │ any IP link: Thunderbolt bridge,
                                    │ Ethernet, USB4, Wi-Fi
┌───────────────────────────────────┴──── machine B (agent role) ──────────────┐
│  connex agent ─┬── mDNS register  (_connex._tcp, ephemeral port)             │
│                ├── auth          (HMAC-SHA256 over a per-connection nonce)   │
│                ├── executor      (spawn process, one goroutine per stream)   │
│                └── framer        (length-prefixed frames, one writev each)   │
└───────────────────────────────────────────────────────────────────────────────┘
```

One binary plays both roles. `connex up` starts the agent; every other command is
a client. A machine is routinely both at once, and the local machine is always a
scheduling candidate scored the same way as any peer.

## Package layout

The vision asks for "a single binary you can read in an afternoon." The best way
to keep it readable at the size connex will reach is a handful of small,
single-purpose internal packages with hard boundaries, not one flat file of
globals. Each package below is intended to be a few hundred lines at most; the
dependency arrows only point downward, so you can read it top to bottom.

```
cmd/connex/            main(): flag parsing, command dispatch, usage, exit codes
internal/
  proto/               wire types, framing (read/write), version constant, errors
  keys/                keyfile load/create, HMAC sign/verify, `key`/`join`
  sysinfo/             cores, total + free memory, load1, gpu/accel  (darwin+linux)
  discovery/           mDNS register (agent) and browse+cache (client), --peer
  agent/               listener, connection handler, executor, signal handling
  transport/           dial + handshake helpers, socket tuning (NODELAY, buffers)
  scheduler/           candidate scoring, filters (--mem/--arch), winner selection
  client/              `ls`, `run`, `each`: prober, stdio pump, exit relay
  paths/               ~/.connex resolution, pidfile, logfile, cache file
```

Rationale and the explicit revision of the original "flat package" line are
recorded in [ADR-0002](adr/0002-internal-package-layout.md). The rule that keeps
it honest: if the whole tree stops fitting in an afternoon's read, a feature is
too big for connex.

## Agent

- Listens on an ephemeral TCP port (`:0`) — no port configuration, no conflicts.
- Advertises `_connex._tcp` over mDNS with TXT metadata (`v`, `os`, `arch`,
  `cpus`). TXT is for identity and coarse filtering only; live load and free
  memory are never in TXT because they would go stale (see Discovery).
- On each connection: send a fresh nonce, verify the client's HMAC, then serve one
  of two ops — `info` (live capacity) or `run` (spawn a process, stream its
  stdio). One operation per connection keeps the protocol a page long
  ([ADR-0005](adr/0005-tcp-length-prefixed-framing.md)).
- Concurrency: one goroutine per accepted connection; within a `run`, one
  goroutine each for stdin→process, process-stdout→client, process-stderr→client,
  plus the main goroutine waiting on the process. No shared mutable state between
  connections, so there are no locks on the accept path.
- `connex up` runs in the foreground; `connex up -d` re-execs itself detached
  (setsid, logs to `~/.connex/agent.log`, writes a pidfile for `connex down`).
  Reboot-persistent launchd/systemd units are roadmap
  ([`CX-OPS-040`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md)).

## Discovery

mDNS/Bonjour, browsing on every up interface. This is the load-bearing
simplification: macOS auto-creates a Thunderbolt Bridge interface with a
link-local address the moment a cable is plugged in, and Linux does the
equivalent for USB/Thunderbolt networking, so wired peer-to-peer links need
*zero* code beyond "browse everywhere." mDNS gives `name → (addresses, port)`.

**Seamlessness through a short cache.** A cold browse takes roughly half a
second, which you would feel on every command. So the client caches discovered
peers at `~/.connex/peers.json` with a short TTL (default ~5 s). The first `run`
pays the browse; the next several are instant. `connex ls` always does a fresh
browse and refreshes the cache. The cache stores only `name → addresses/port`,
never capacity — liveness always comes from a fresh `info` probe, so a cached but
now-busy peer is never scheduled onto blindly. Details and the tradeoff against
"no client state at all" are in [ADR-0003](adr/0003-mdns-discovery-and-cache.md).

**Explicit peers.** `--peer host:port` (and a `~/.connex/peers` allowlist) skips
mDNS entirely, which makes connex work across a Tailscale tailnet or any network
where multicast is filtered ([`CX-DISC-020`](../roadmaps/milestones/m2-v0.2-real-use.md)).

## Scheduler

Deliberately dumb in v1:

```
score(machine) = load1 / cpu_count        # lower is better, local included
```

Every discovered peer is probed for live `info` in parallel with a tight 250 ms
budget; a peer that times out or answers `unauthorized` is dropped from the
candidate set, never retried inline. The local machine is always a candidate and
is scored from local sysinfo with no network cost. If it wins — or if there are no
peers — the command execs locally and connex prints nothing: the invisible
fallback. `--on <name>` pins a target and skips scoring.

Planned refinements, in value order (v0.3+), each a filter applied before scoring:

- **Memory-aware placement**: `--mem 48` filters to machines whose reported
  `mem_free_gb` clears the bar — critical for model loading.
- **Link preference**: when a peer resolved on multiple interfaces, prefer the
  wired/bridge address; tie-break scoring toward the faster link.
- **Arch constraints**: skip cross-arch targets for native binaries; `--arch
  arm64` to filter.

## Execution model

The remote process runs as the agent's user, in the agent's home directory, with
the agent's environment. Data path:

- **stdin** streams from the client unframed and is spliced into the process's
  stdin; the client half-closes its TCP write side on stdin EOF, and the agent
  propagates that EOF to the process.
- **stdout/stderr** come back as typed length-prefixed frames so they demux to the
  right local descriptors without corrupting binary output. Each frame is written
  with a single `writev` (header + payload coalesced) and the payload buffers are
  pooled, so steady-state streaming does no per-frame allocation.
- **exit code** is carried in a final frame and becomes the client's exit code.

This is what makes `connex run` pipe-safe and script-safe. The end-to-end latency
and throughput targets for this path are the subject of
[10-performance.md](10-performance.md).

Not yet carried across (roadmap): working-directory + file sync
([`CX-EXEC-030`](../roadmaps/milestones/m2-v0.2-real-use.md)), TTY allocation for
interactive programs, and full signal forwarding. v0.1 already forwards SIGINT so
Ctrl-C on the client reaches the remote process and leaves no orphan
([`CX-EXEC-010`](../roadmaps/milestones/m1-v0.1-fabric.md)).

## Process and data lifecycle (a `run` end to end)

```
1. client: load key; read peer cache (or browse mDNS if stale)      [disc]
2. client: parallel `info` probe of candidates, 250 ms budget       [prober]
3. client: score candidates + local; pick winner                    [scheduler]
4. winner == local?  → exec locally, inherit stdio, exit. print nothing.
5. else: dial winner, set TCP_NODELAY                               [transport]
6. server→client: "CONNEX1 <nonce>\n"                               [proto]
7. client→server: {hmac, op:"run", cmd:[...]}\n                     [proto]
8. server: verify HMAC; spawn process; reply {"ok":true}\n          [agent]
9. client: print one dim "connex → <name>" line to stderr           [client]
10. bidirectional stream: stdin↑ (raw), stdout/stderr↓ (framed)     [framer]
11. process exits → server sends exit frame → client exits w/ code
```

Steps 1–3 are the only ones that can add perceptible latency, and the cache plus
the parallel-probe budget bound them. Steps 5–11 run at link speed.
