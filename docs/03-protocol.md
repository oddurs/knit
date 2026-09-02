# Wire protocol

Version token `CONNEX1`. One TCP connection per operation. Small enough to fit on
a page, by design — that page is this document.

## Design goals

1. **One page.** A reader implements a client from this doc alone.
2. **One round-trip to first byte.** After the connection is open, exactly one
   server line and one client line precede streaming; no negotiation ping-pong.
3. **Binary-clean and pipe-safe.** stdout/stderr are framed so arbitrary bytes
   (including NULs and the frame bytes themselves) pass through untouched.
4. **Cheap to serve.** Header is fixed-size; the agent never buffers a whole
   stream; steady state does no per-frame allocation.

## Socket setup

Both ends set `TCP_NODELAY` (Nagle off) immediately after connect/accept — connex
sends many small frames interactively, and Nagle would add up to 40 ms of
head-of-line delay per turn. Send/receive buffers are left to the OS default on
LAN and raised on high-bandwidth-delay links only if benchmarks show a ceiling
([10-performance.md](10-performance.md)). No keepalive in v1: connections are
short-lived and a drop is meaningful (see below).

## Handshake

```
server → client:   CONNEX1 <32-hex-char nonce>\n
client → server:   {"v":1,"hmac":"<hex hmac-sha256(key,nonce)>","op":"run","cmd":["zstd","-19"]}\n
server → client:   {"ok":true}\n                       (or {"ok":false,"error":"...","code":"..."}\n)
```

- The nonce is 16 random bytes, hex-encoded, fresh per connection. The HMAC is
  `HMAC-SHA256(key, nonce)`, hex-encoded. The key never crosses the wire; replay
  is useless because the nonce is single-use.
- Verification is constant-time. A failed verification gets exactly one
  `{"ok":false,...}` line with code `unauthorized`, then the connection closes.
- The client sends `"v":1`. If a future agent only speaks a higher version it
  replies `{"ok":false,"code":"version"}` and closes; the client prints a
  fix-naming error. Unknown JSON fields are ignored in both directions, so minor
  additions never break older peers.

Ops: `info` and `run`.

## op = info

After a successful handshake for `op:"info"`, the server sends one JSON envelope
and closes:

```json
{"ok":true,"name":"studio","os":"darwin","arch":"arm64","cpus":24,
 "mem_gb":128.0,"mem_free_gb":96.4,"load1":0.42,"accel":"metal","gpu":"Apple M3 Ultra"}
```

`mem_free_gb`, `accel`, and `gpu` ship in v0.3 ([`CX-SYS-030`](../roadmaps/milestones/m3-v0.3-ai-native.md));
v0.1 populates `name/os/arch/cpus/mem_gb/load1`. Consumed by `connex ls`, the
scheduler, and — because it requires a valid HMAC — as the authentication probe.

## op = run

After `{"ok":true}`:

- **client → server**: raw stdin bytes, unframed. The client half-closes the TCP
  write side (`CloseWrite`) on stdin EOF; the agent propagates EOF to the process.
- **server → client**: length-prefixed frames.

```
frame = [type:1][len:4 big-endian][payload:len]

type 1  stdout chunk
type 2  stderr chunk
type 3  exit      payload = 4-byte big-endian exit code; connection closes after
type 4  signal    (reserved, CONNEX2 — client→server needs both directions framed)
type 5  winsize   (reserved, CONNEX2 — TTY support)
```

- `len` is capped at 1 MiB (`maxFrame`); a larger declared length is a protocol
  violation and the receiver closes. Chunks are whatever the reader produced,
  typically ≤ the 32–256 KiB pump buffer, never split across frames of the wrong
  type.
- The agent writes each frame with a single vectored write (header + payload) to
  avoid a syscall per part, and reuses payload buffers from a pool.
- **Termination:** a clean run ends with an `exit` frame carrying the process's
  code, which becomes the client's exit code. A connection that drops *before* the
  exit frame is an error: the client exits non-zero (code `disconnected`) and the
  agent reaps the process when its stdio closes. There is no ambiguity between
  "exited 0" and "link died," because 0 only ever arrives inside an explicit exit
  frame.

## Error codes

Envelope errors carry a stable machine `code` alongside human `error` text, so the
CLI can name the fix (principle: errors name the fix, not just the failure).

| code           | meaning                                   | CLI guidance shown |
| -------------- | ----------------------------------------- | ------------------ |
| `unauthorized` | HMAC did not verify                       | run `connex key` on a trusted machine, `connex join <key>` here |
| `version`      | agent speaks a newer protocol             | upgrade connex on this machine |
| `empty_cmd`    | `run` with no command                     | (client-side guard; should not reach the wire) |
| `spawn`        | process failed to start (e.g. not found)  | the underlying exec error, verbatim |
| `internal`     | agent-side unexpected error               | check `~/.connex/agent.log` on the target |

## Forward compatibility: CONNEX2 (sketch, not v1)

Signals and TTYs both require the **client→server** direction to be framed too, so
they are gated behind a version bump rather than bolted onto CONNEX1. CONNEX2
keeps the handshake identical and changes only the post-`ok` run phase: both
directions carry frames, adding `signal` (SIGINT/SIGTERM/SIGWINCH) and `winsize`.
v0.1 gets the 90% win — Ctrl-C reaching the remote process — by sending a single
out-of-band SIGINT as a one-byte control before `CloseWrite`, documented in
[`CX-EXEC-010`](../roadmaps/milestones/m1-v0.1-fabric.md); full framing waits.

## Security summary

Trust is symmetric key possession; `run` is arbitrary code execution by design, so
authentication is the whole game. The full threat model, the accepted v1 gaps
(no transport encryption yet), and the TLS-with-pinned-certs plan live in
[08-security-model.md](08-security-model.md).
