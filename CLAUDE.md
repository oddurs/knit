# CLAUDE.md

Guidance for AI assistants (and humans) working in this repository.

## What knit is

A single static Go binary that lets machines on the same desk, cable, or LAN
share compute with zero configuration. `knit up` makes a machine discoverable
capacity; `knit run -- <cmd>` executes on whichever machine has the most spare
headroom, streaming stdio as if it ran locally. Read `docs/` before changing
behavior — the design is deliberate and documented.

## North star (in priority order)

1. **Simple** — one binary, one runtime dependency (mDNS), seven commands, a wire
   protocol that fits on a page. If a feature needs a config file or an eighth
   command, it waits or dies.
2. **Fast** — the streaming path is line-rate; the hot path allocates nothing per
   frame (`go test -bench . -benchmem ./internal/proto` must stay 0 allocs/op).
3. **Seamless** — exit codes, stdin EOF, and pipes behave byte-for-byte as local.
   `knit run` prints one dim line only when work left the machine.

## Ground rules

- **Never add a Claude co-author trailer, "Generated with" line, or any AI
  attribution to commits or PRs.** Commits are authored by the repository owner.
  Keep messages plain and factual.
- Keep the dependency count at one. A new module dependency needs an ADR
  (`docs/adr/`) and must remove more complexity than it adds.
- OS-specific code lives only in `internal/sysinfo` behind build tags. Everything
  else is portable.
- The plan is code: work items live in `roadmaps/registry.toml` with stable IDs
  (`KN-<AREA>-<NNN>`). Validate with `python3 roadmaps/tools/roadmap.py check` and
  re-render `STATUS.md` after changes.

## Workflow

```sh
go build ./...                 # build
go test -race ./...            # test (race detector is non-negotiable)
gofmt -l cmd internal          # must be empty
go vet ./...                   # must be clean
python3 roadmaps/tools/roadmap.py check   # the plan must stay valid
```

Build a release binary:

```sh
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(git describe --tags --always)" -o knit ./cmd/knit
```

## Layout

- `cmd/knit` — entrypoint, dispatch, exit-code contract.
- `internal/` — `proto` (wire), `keys` (auth), `sysinfo`, `discovery` (mDNS+cache),
  `transport` (dial+handshake), `agent` (server+exec), `scheduler`, `client`,
  `paths`.
- `docs/` — vision, architecture, protocol, security, performance, ADRs.
- `roadmaps/` — the coded plan and its validator.
