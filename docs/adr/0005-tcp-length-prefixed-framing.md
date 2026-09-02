# ADR-0005: Raw TCP + length-prefixed frames, one op per connection
Status: Accepted
Date: 2026-09-01

## Context
The data path must stream stdin/stdout/stderr binary-clean at link rate, demux the
two output streams, and carry an exit code unambiguously — while keeping the
protocol to one page and the dependency count at zero for the transport.

## Decision
Raw TCP, `TCP_NODELAY` on. One operation per connection. Control messages are one
JSON line each (handshake, `info`). The `run` data plane is length-prefixed frames
`[type:1][len:4 BE][payload]` with types stdout/stderr/exit (signal/winsize
reserved for CONNEX2). Frames cap at 1 MiB; each frame is one `writev`. Detailed in
[03-protocol.md](../03-protocol.md).

## Alternatives considered
- **gRPC / HTTP/2** — multiplexing and flow control connex doesn't need on a single
  short LAN stream, plus a large dependency and codegen. Costs the one-page protocol.
- **QUIC** — congestion control and 0-RTT are wasted on a point-to-point cable;
  heavy dependency.
- **SSH transport** — re-uses a trusted protocol but drags in host-key UX, config,
  and a dependency, and fights the zero-config posture.
- **Newline-delimited or base64 stdout** — not binary-clean; corrupts arbitrary
  output. Framing is required.
- **One long-lived multiplexed connection per peer** — would save handshakes but
  needs stream IDs and flow control, breaking the one-page rule. Connection reuse is
  a tracked backlog optimization instead ([CX-XPORT-050](../../roadmaps/areas/backlog.md)).

## Consequences
- The protocol fits on a page and needs no transport dependency.
- Exit `0` is unambiguous: it only ever arrives inside an explicit exit frame, so a
  dropped link (client exits `124`) is never confused with success.
- One `writev` per frame + pooled buffers keep the data path allocation-free.
- Per-op connections cost one handshake each; accepted for simplicity, with reuse
  parked as a measured future win.
