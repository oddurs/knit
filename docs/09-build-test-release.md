# Build, test, and release

## Build

One command, no CGo, reproducible:

```sh
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X main.version=$(git describe --tags --always)" \
  -o knit ./cmd/knit
```

- `CGO_ENABLED=0` → a fully static binary; `sysinfo` uses `/proc` and `sysctl`
  (via `os/exec`), never libc, so nothing pulls CGo back in.
- `-trimpath` strips local paths for reproducibility.
- Cross-compile the four targets by setting `GOOS`/`GOARCH`; no toolchain beyond
  the Go compiler is required.

## Test strategy

Fast, hermetic, and honest — tests must not depend on a real second machine to run
in CI, but the two-machine reality is verified by a documented manual smoke test.

| Layer | What | How |
| ----- | ---- | --- |
| **unit** | framing round-trips, HMAC sign/verify, keyfile load/create, scheduler scoring, sysinfo parsers | table tests, golden inputs; parsers fed captured `/proc` and `sysctl` fixtures |
| **protocol** | full handshake + `info` + `run` over an in-process `net.Pipe` and over loopback TCP | a real agent goroutine, a real client, asserting bytes and exit codes |
| **loopback e2e** | `knit up -d`, `knit ls`, `echo hi \| knit run --on $(hostname -s) -- cat` | scripted; runs on one machine, exercises the whole stack against the local agent |
| **race** | the pump/exit goroutines | `go test -race ./...` in CI, non-negotiable |
| **fuzz** | frame reader (`len` bounds, truncation) and the JSON handshake parser | `go test -fuzz`, seeded corpus committed |
| **manual matrix** | two machines over Thunderbolt bridge and over Wi-Fi | [`manual-test.md`](manual-test.md) checklist; run before each tagged release |

Coverage is a signal, not a target; the framing, auth, and exit-code paths are the
ones that must be near-total because they are where a bug corrupts data or lies
about success.

## Benchmarks (guarding the north star)

Speed is a tracked property, not a vibe. `go test -bench` covers:

- `BenchmarkFrameWrite` — allocations per frame must be **0** in steady state
  (asserted via `-benchmem`, fails CI if it regresses above the pooled baseline).
- `BenchmarkPumpThroughput` — bytes/sec through the stdout pump over loopback;
  tracked over time to catch a copy-path regression.
- A manual `iperf`-style throughput check over a real Thunderbolt cable, recorded
  in [10-performance.md](10-performance.md) so the "line rate" claim has a number
  behind it.

## Continuous integration

GitHub Actions, on push and PR:

1. `gofmt -l` (must be empty) and `go vet ./...`
2. `staticcheck ./...`
3. `go test -race ./...`
4. `go build` for all four `GOOS/GOARCH` targets (compile-only cross-check)
5. `python3 roadmaps/tools/roadmap.py check` — the plan itself must stay valid:
   unique IDs, resolvable dependencies, no cycles ([`KN-TEST-002`](../roadmaps/milestones/m1-v0.1-fabric.md))

## Release

- Tag `vMAJOR.MINOR.PATCH`; `goreleaser` builds the four binaries, checksums, and
  a source archive, and cuts a GitHub release.
- A `knit` Homebrew tap formula points at the release
  ([`KN-OPS-021`](../roadmaps/milestones/m2-v0.2-real-use.md)); `brew install
  knit/tap/knit` is the headline install path.
- `knit --version` prints the `-ldflags`-injected version so bug reports are
  precise.
- Versioning follows the milestone line (v0.1 → v0.4 → v1.0); the wire protocol
  has its own token (`KNIT1`) that only bumps on an incompatible change, decoupled
  from the CLI version.
