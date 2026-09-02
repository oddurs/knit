# Performance

Speed is a feature of connex, designed in rather than tuned in later. This doc is
the budget every change is measured against and the set of techniques that meet
it. The governing idea: **on the hot path, the physical link is the only thing
allowed to be the bottleneck.**

## The two paths

connex has a *control path* (discovery, probing, scheduling, handshake) and a
*data path* (streaming stdio). They have completely different budgets:

- The **control path** runs once per command and is allowed to cost a bounded,
  small amount of wall-clock — tens of milliseconds warm, a few hundred cold. It
  must never cost the user a *second* stall for the *second* command.
- The **data path** runs for the whole life of the command and must add
  near-zero overhead over a raw socket copy. No allocation, no extra syscalls, no
  double-buffering.

## Latency budget (control path)

Numbers are targets on a quiet LAN / Thunderbolt bridge; they are what the
benchmarks and manual tests defend.

| Stage                          | Cold        | Warm (cached) | Technique |
| ------------------------------ | ----------- | ------------- | --------- |
| Peer discovery                 | ~300–600 ms | ~0 ms         | mDNS browse, then ~5 s peer cache |
| Parallel `info` probe          | ≤ 250 ms    | ≤ 250 ms      | all peers dialed at once, hard deadline, slow peers dropped |
| Scheduling decision            | < 1 ms      | < 1 ms        | arithmetic over a handful of structs |
| Local-win fast path            | 0 network   | 0 network     | if local scores best, exec directly — no dial at all |
| Handshake to first byte (remote) | 1 RTT     | 1 RTT         | one server line + one client line, no negotiation |

**Time-to-first-byte for a remote run, warm:** roughly one probe budget + one RTT
— low tens of milliseconds on a cable. For a local win it is the cost of `exec`,
i.e. indistinguishable from typing the command yourself.

### Making the second command instant

A cold mDNS browse is the single largest control-path cost, and paying it on every
command would make connex feel laggy. So the client caches `name → addr:port`
(never capacity) at `~/.connex/peers.json` with a short TTL (~5 s, `CONNEX_NO_CACHE`
to disable, `CONNEX_TIMEOUT_MS` to tune the probe). Back-to-back commands — the
common case in a build loop or an inference session — skip the browse entirely and
re-probe only for live load. Freshness of *placement* is preserved because load is
always probed live; only *addresses* are cached, and a stale address simply fails
the probe and is dropped.

## Data-path techniques (the blazing-fast internals)

1. **Nagle off.** `TCP_NODELAY` on both ends. Interactive, frame-at-a-time traffic
   would otherwise eat up to 40 ms per turn to Nagle's algorithm. This single
   setting is the difference between "feels remote" and "feels local."

2. **One `writev` per frame.** The framer writes `[header][payload]` as a
   `net.Buffers{hdr, payload}`, which Go lowers to a single `writev(2)` syscall.
   Two writes per frame would double syscall count on the busiest path; we do one.

3. **Zero steady-state allocation.** Pump buffers (32–256 KiB) are taken from a
   `sync.Pool` and reused; frame headers are stack-allocated fixed arrays. A
   `-benchmem` test asserts **0 allocs/op** for `FrameWrite` in steady state and
   fails CI if that regresses. The GC therefore never touches the hot path, which
   is why Go's GC is a non-issue here.

4. **Right-sized copies, OS zero-copy where offered.** The stdout/stderr pumps use
   `io.CopyBuffer` with a pooled buffer. On Linux, socket↔pipe copies can engage
   `splice(2)` (kernel-side, no userspace copy) for eligible fd pairs; we shape the
   plumbing so the kernel can take that path and measure whether it does. stdin
   (client→process) is a straight `io.Copy` from the connection into the process
   pipe, which the kernel can likewise optimize.

5. **No serialization on the data plane.** Bytes are framed, never encoded. JSON
   appears exactly twice per connection (handshake + `info`), both off the hot
   path.

6. **Bounded, non-blocking control.** Frame size caps at 1 MiB and probes have a
   hard 250 ms deadline, so one slow or malicious peer can never stall the pump or
   the scheduler.

## Throughput ceiling

With Nagle off, one `writev` per frame, and pooled buffers, the client's CPU cost
per byte is a single `memmove`-class copy (or none, when `splice` engages). The
practical ceiling is therefore:

- **Over Wi-Fi:** the radio (hundreds of Mbit/s to low Gbit/s) — connex is never
  the limit.
- **Over Ethernet:** the NIC line rate.
- **Over Thunderbolt 4/5 bridge:** many GB/s; here the memory-copy path can
  matter, which is exactly why the pump avoids double-buffering and prefers the
  kernel splice path. The manual benchmark records the achieved number so the
  "line rate" claim in the vision is backed by data, not adjectives.

## What we deliberately do *not* optimize (yet)

- **Connection reuse across ops.** v1 opens one connection per op for protocol
  simplicity; the `info` probe connection is not upgraded into the following
  `run`. Reusing it would shave one dial+handshake off a remote run. It is a real
  win and a real complication, so it is a tracked backlog item
  ([`CX-XPORT-050`](../roadmaps/areas/backlog.md)), not a v1 feature — simplicity
  first, and the saved RTT is small next to the command's own runtime.
- **A client-side discovery daemon.** Would make discovery truly 0 ms always, at
  the cost of a second long-lived process and stale-state bugs. The 5 s cache
  captures most of the benefit for none of the cost. Revisited only if the cache
  proves insufficient in practice.

## How performance is defended

Every claim here maps to a check in [09-build-test-release.md](09-build-test-release.md):
`-benchmem` guards the 0-alloc invariant, `BenchmarkPumpThroughput` guards the copy
path, and the manual Thunderbolt run guards the end-to-end number. A change that
regresses any of them does not merge without a recorded, deliberate reason.
