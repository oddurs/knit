# Wire protocol

Version token `KNIT1`. One TCP connection per operation. Small enough to fit on
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

Both ends set `TCP_NODELAY` (Nagle off) immediately after connect/accept — knit
sends many small frames interactively, and Nagle would add up to 40 ms of
head-of-line delay per turn. Send/receive buffers are left to the OS default on
LAN and raised on high-bandwidth-delay links only if benchmarks show a ceiling
([10-performance.md](10-performance.md)). No keepalive in v1: connections are
short-lived and a drop is meaningful (see below).

## Handshake

The TCP connection is wrapped in TLS 1.3 first (ephemeral self-signed server
certificate, not verified by the client). Everything below travels inside it.
Both ends derive a 32-byte *channel binding* from the TLS keying material
(`ExportKeyingMaterial("knit channel binding")`), and every proof includes it.

```
server → client:   KNIT1 <32-hex-char nonce>\n
client → server:   {"v":3,"hmac":"<hex hmac-sha256(key,nonce)>","op":"run","cmd":["zstd","-19"]}\n
                   optional: "dir":true, "sync":true (working-directory transfer);
                             "hosts":[addr,...], "rank":n (a `knit each` launch: the
                             agent exports KNIT_RANK/NNODES/HOSTS/MASTER and
                             MLX_RANK/MLX_HOSTFILE to the command)
server → client:   {"ok":true,"proof":"<hex>"}\n        (or {"ok":false,"error":"...","code":"..."}\n)
```

- The nonce is 16 random bytes, hex-encoded, fresh per connection. The client's
  `hmac` is `HMAC-SHA256(key, "knit client" ‖ 0 ‖ nonce ‖ 0 ‖ binding)`; the
  server's `proof` is the same over `"knit server"`. The key never crosses the
  wire; replay is useless because the nonce is single-use, and a proof relayed
  through a machine in the middle fails because the binding differs per leg.
- Verification is constant-time on both sides. A failed client proof gets
  exactly one `{"ok":false,...}` line with code `unauthorized`, then the
  connection closes. A missing or wrong server proof makes the client close
  and report the agent as not holding the key.
- The client sends its protocol version in `"v"` (currently `3`, the TLS
  generation). A v3 client reaching a plaintext v2 agent sees a TLS record
  error and reports "an older knit"; unknown JSON fields are ignored in both
  directions, so minor additions never break peers within a generation.

Ops: `info`, `run`, and `dial`. A connection is authenticated once; an `info`
probe may be followed by a `run` on the same connection, so `knit run` reuses
the probe it just did instead of dialing again ([`KN-XPORT-050`](../roadmaps/registry.toml)).
An agent too old to accept the follow-up simply closes, and the client dials afresh.

## op = info

After a successful handshake for `op:"info"`, the server sends one JSON envelope
and closes:

```json
{"ok":true,"name":"studio","os":"darwin","arch":"arm64","cpus":24,
 "mem_gb":128.0,"mem_free_gb":96.4,"load1":0.42,"accel":"metal","gpu":"Apple M3 Ultra"}
```

`mem_free_gb` is what could be allocated now (macOS: total minus wired,
compressed, and app memory, as Activity Monitor counts it; Linux:
`MemAvailable`). `accel` is `metal`, `cuda`, or `none`; `gpu` names the chip or
card when there is one. Consumed by `knit ls`, the scheduler, and — because it
requires a valid HMAC — as the authentication probe.

## op = run

After `{"ok":true}` both directions carry length-prefixed frames (KNIT2, `v>=2`).
The two directions use disjoint type numbers so they never collide in a log.

```
frame = [type:1][len:4 big-endian][payload:len]

server → client
  type 1   stdout chunk
  type 2   stderr chunk
  type 3   exit       payload = 4-byte big-endian exit code; connection closes after

client → server
  type 10  stdin chunk
  type 11  stdin EOF  payload empty; the agent closes the process's stdin
  type 12  signal     payload = 1 byte signal number (2 = SIGINT, 15 = SIGTERM)
  type 13  winsize    reserved for TTY support
```

Framing the client direction is what lets Ctrl-C forward the *actual* signal to
the remote process — which can trap it and exit with its own code — and makes a
piped stdin's EOF (`type 11`) unambiguous, distinct from the client vanishing.

- `len` is capped at 1 MiB (`maxFrame`); a larger declared length is a protocol
  violation and the receiver closes. Chunks are whatever the reader produced,
  typically ≤ the 32–256 KiB pump buffer, never split across frames of the wrong
  type.
- The agent writes each frame with a single vectored write (header + payload) to
  avoid a syscall per part, and reuses payload buffers from a pool.
- **Termination:** a clean run ends with an `exit` frame carrying the process's
  code, which becomes the client's exit code. If the client's frame stream ends
  before the exit frame was sent — the connection dropped, the client was
  killed, with or without a stdin-EOF first — the agent reaps the whole process
  group (SIGINT, then SIGKILL after 2 s), so nothing is orphaned. A client that
  loses the link before the exit frame exits non-zero (code `disconnected`).
  There is no ambiguity between "exited 0" and "link died," because 0 only ever
  arrives inside an explicit exit frame.

## Error codes

Envelope errors carry a stable machine `code` alongside human `error` text, so the
CLI can name the fix (principle: errors name the fix, not just the failure).

| code           | meaning                                   | CLI guidance shown |
| -------------- | ----------------------------------------- | ------------------ |
| `unauthorized` | HMAC did not verify                       | run `knit key` on a trusted machine, `knit join <key>` here |
| `version`      | agent speaks a newer protocol             | upgrade knit on this machine |
| `empty_cmd`    | `run` with no command                     | (client-side guard; should not reach the wire) |
| `spawn`        | process failed to start (e.g. not found)  | the underlying exec error, verbatim |
| `internal`     | agent-side unexpected error               | check `~/.knit/agent.log` on the target |

## Versioning

`v2` (KNIT2) framed the client→server direction, adding stdin/stdin-EOF/signal
frames as above ([`KN-EXEC-020`](../roadmaps/milestones/m2-v0.2-real-use.md)).
`v3` wrapped the connection in TLS and bound both proofs to it
([`KN-AUTH-040`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md)); it
is the first version boundary that does not interoperate with the previous
one, and each side says so. A future version would add `winsize` (type 13)
and a TTY mode for interactive programs.

## op = dial

`knit proxy` tunnels raw TCP through a peer. After a handshake for
`op:"dial","host":"ip:port"`, the agent connects to that address, replies
`{"ok":true,"proof":"..."}`, and then splices bytes in both directions over the
same TLS connection — no framing, just an encrypted, authenticated pipe. Only a
key holder reaches it, so it is arbitrary outbound connection on the peer's
behalf, the same trust `run` already grants
([`KN-NET-040`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md)).

## Security summary

Trust is symmetric key possession; `run` is arbitrary code execution by design, so
authentication is the whole game. The link is TLS 1.3 authenticated by that key
in both directions. The full threat model and its accepted gaps live in
[08-security-model.md](08-security-model.md).
