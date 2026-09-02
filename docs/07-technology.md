# Technology choices

Every choice below is scored against the north star — **simple, fast, seamless,
in that order** — and each irreversible one has an ADR. This doc is the map.

## Language: Go

Go is the right substrate for knit, and the reasoning is in
[ADR-0001](adr/0001-language-go.md). In brief:

- **Single static binary, trivially cross-compiled** for darwin/arm64,
  darwin/amd64, linux/arm64, linux/amd64 from one machine. This is the whole
  distribution story — `scp` the binary, or `brew install`, and you are done.
- **Concurrency that matches the problem.** The execution model is "a few
  goroutines per connection moving bytes between a socket and a pipe." Goroutines
  + channels express it in a few dozen lines with no callback soup.
- **A standard library that already contains knit's hard parts:** `crypto/hmac`,
  `crypto/rand`, `net` with `TCP_NODELAY` and `CloseWrite`, `os/exec` with pipe
  plumbing, `encoding/binary` and `encoding/json`. The runtime dependency count is
  therefore *one*.
- **Fast enough that the network is always the bottleneck.** Go's `io.Copy`
  transparently uses `splice`/`sendfile` on Linux for eligible fd pairs, escape
  analysis keeps pump buffers off the heap, and `net.Buffers` gives us a single
  `writev` per frame. See [10-performance.md](10-performance.md).
- **Readable by the widest audience** — the vision's "read in an afternoon" test.

Rust would give marginally lower tail latency and no GC, at the cost of a slower
build-and-read experience and more ceremony for the same byte-moving code. For a
tool whose bottleneck is a physical cable, that trade is not worth it. Revisit
only if profiling ever shows the GC on the hot path (it will not — the hot path
does no allocation).

## Runtime dependency: exactly one (mDNS)

`github.com/grandcat/zeroconf` for multicast-DNS register + browse. Rationale and
the fallback plan (vendoring a minimal browser, or `hashicorp/mdns`) are in
[ADR-0003](adr/0003-mdns-discovery-and-cache.md). Everything else is stdlib.

**Dependency policy:** a new module dependency is a design smell that needs an ADR
to land. The bar: it must remove more complexity than it adds, be a single
well-scoped package, and have no transitive sprawl. `go.sum` is committed; `go mod
tidy` is clean in CI; Dependabot-style bumps are reviewed, not automatic.

## Transport: raw TCP (v1), TLS 1.3 over TCP (v0.4)

- v1 is plaintext TCP on the trusted local link — simplest thing that streams at
  line rate. `TCP_NODELAY` on; OS-default buffers; one connection per op.
- v0.4 wraps the same TCP in TLS 1.3 with per-machine self-signed certs whose
  fingerprints are pinned at `join` time — keeps zero-config while closing the
  plaintext gap. TLS 1.3 adds one round-trip to the handshake and negligible
  streaming cost with AES-NI/hardware crypto. See
  [08-security-model.md](08-security-model.md).
- **Not QUIC/HTTP/gRPC.** They buy multiplexing and congestion control knit does
  not need on a single short-lived LAN stream, and they cost the "one page"
  protocol and a pile of dependencies. Recorded in
  [ADR-0005](adr/0005-tcp-length-prefixed-framing.md).

## Serialization: JSON for control, raw bytes for data

The handshake and `info` envelope are one JSON line each — human-debuggable with
`nc`, forward-compatible via ignored unknown fields, and utterly not on the hot
path. The data plane is raw framed bytes, never JSON, so throughput is never
gated by a serializer. No protobuf/msgpack: they would add a dependency and a
codegen step to save nothing on messages that occur once per connection.

## Auth crypto: HMAC-SHA256 over a per-connection nonce

Symmetric, in the stdlib, constant-time compare, key never on the wire, replay
useless. Full reasoning in [ADR-0004](adr/0004-hmac-nonce-auth.md).

## Discovery: mDNS + a short client-side cache

mDNS for zero-config `name → addr:port`; a ~5 s peer cache so back-to-back
commands feel instant without a client daemon. Tradeoff in
[ADR-0003](adr/0003-mdns-discovery-and-cache.md). `--peer` / `KNIT_PEERS` is the
escape hatch where multicast is filtered (Tailscale, some corporate LANs).

## Build & release: `go build`, `goreleaser`, a Homebrew tap

Reproducible static builds via `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w
-X main.version=..."`, cross-compiled in CI for the four targets, released with
`goreleaser`, distributed via a `knit` Homebrew tap and raw binary attachments.
Details in [09-build-test-release.md](09-build-test-release.md).

## Platform support

| Platform       | Agent | Client | Notes                                            |
| -------------- | ----- | ------ | ------------------------------------------------ |
| macOS arm64    | ✓     | ✓      | primary target; Thunderbolt Bridge is the story  |
| macOS amd64    | ✓     | ✓      | older Intel Macs                                 |
| Linux arm64    | ✓     | ✓      | `/proc` for load/mem; USB4/TB networking         |
| Linux amd64    | ✓     | ✓      | NVML for GPU (v0.3)                              |
| Windows        | —     | maybe  | client conceivable; agent needs job-object work not worth it yet |

sysinfo is the only OS-specific code (`internal/sysinfo`), split by build tag per
OS. Everything else is portable Go.
