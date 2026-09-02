# 🧶 knit

**Share compute across your machines with zero config.** One static binary. Run
`knit up` on each machine and it becomes discoverable capacity. Run `knit run
-- <cmd>` anywhere and it executes on whichever machine — including this one — has
the most spare headroom, with stdin, stdout, stderr, and the exit code behaving
byte-for-byte as if it ran locally.

```console
$ knit up            # on each machine, once
$ knit run -- ffmpeg -i big.mov out.mp4
knit → studio        # ran on the Mac Studio; output streamed back here
```

No IPs, no config files, no accounts, no server. A Thunderbolt cable, Ethernet,
USB4, and Wi-Fi all work the same way — knit discovers peers on every interface,
and the cable is just the fastest one.

## Why

Every desk with more than one computer wastes most of them. Wiring them together
today means SSH configs, IP addresses, and hostfiles. knit is the missing
fabric: find machines, know their capacity, put a process on one (or all) with
streamed I/O — and nothing more. Distributed inference launchers (MLX, llama.cpp
RPC, exo) run on top with one command instead of a page of setup.

## Install

```sh
# macOS
brew install oddurs/tap/knit

# Any platform, from source (Go 1.26+)
go install github.com/oddurs/knit/cmd/knit@latest
```

Linux binaries for amd64 and arm64 are on the
[releases page](https://github.com/oddurs/knit/releases).

## Quickstart

```sh
# 1. Start the agent on each machine (background)
knit up -d

# 2. Pair a second machine, one time
knit key                 # on machine A → prints 64 hex chars
knit join <key>          # on machine B

# 3. Use the fabric
knit gauge               # see machines and their capacity
knit run -- make -j      # runs where there's most headroom
knit run --on studio -- python quantize.py
cat data.csv | knit run -- zstd -19 > data.zst
knit each -- uname -a    # run everywhere at once
```

## Commands

| Command | Does |
| ------- | ---- |
| `knit up [-d]` | start sharing this machine (`-d`: background) |
| `knit down` | stop the background agent |
| `knit gauge [--json]` | show machines and their capacity (alias: `ls`) |
| `knit run [--on NAME] -- CMD` | run where there's most headroom |
| `knit each -- CMD` | run on every machine at once |
| `knit key` | print this machine's cluster key |
| `knit join KEY` | join the fabric that key belongs to |

**A little nomenclature.** All your machines together are the **fabric**, one
machine is a **loop**, and **gauge** is a machine's capacity. That's it — the rest
is plain words.

## How it works

- **Discovery** — multicast-DNS on every interface (`_knit._tcp`), with a short
  client-side cache so back-to-back commands are instant.
- **Auth** — machines sharing a 32-byte key trust each other. Each connection is
  authenticated with an HMAC over a fresh server nonce; the key never crosses the
  wire. `run` is arbitrary code execution by design, so trust is the whole game.
- **Scheduling** — score each machine as load-per-core and pick the lowest; the
  local machine is always a candidate, and if it wins, the command runs locally
  and knit prints nothing.
- **Streaming** — one TCP connection per command, `TCP_NODELAY` on, output framed
  so it is binary-clean; the hot path allocates nothing per frame.

Full design: [`docs/`](docs/). The staged plan and its validator: [`roadmaps/`](roadmaps/).

## Security

Trusted local links only. v1 authenticates but does not yet encrypt the link, so
use it on a Thunderbolt cable or a home LAN, not a hostile network. Transport
encryption (TLS 1.3 with pinned certs) is planned for v0.4. See
[`docs/08-security-model.md`](docs/08-security-model.md) and [`SECURITY.md`](SECURITY.md).

## Status

v0.1 (the fabric) is functionally complete. `up`/`down`/`ls`/`run`/`each`/`key`/`join`
work end to end; stdin/stdout/stderr stream binary-clean, exit codes are relayed,
and Ctrl-C reaps the remote process with no orphan. The whole suite — unit,
in-process protocol e2e, and the process-reaping test — runs under `-race`, and
the frame hot path is guarded at zero allocations per frame. The one remaining
v0.1 item is the two-machine manual matrix over a real Thunderbolt link and Wi-Fi
(see [`docs/manual-test.md`](docs/manual-test.md)); it needs a second machine to
run. Track everything in [`roadmaps/STATUS.md`](roadmaps/STATUS.md).

## License

MIT — see [`LICENSE`](LICENSE).
