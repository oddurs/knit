# connex

**Share compute across your machines with zero config.** One static binary. Run
`connex up` on each machine and it becomes discoverable capacity. Run `connex run
-- <cmd>` anywhere and it executes on whichever machine — including this one — has
the most spare headroom, with stdin, stdout, stderr, and the exit code behaving
byte-for-byte as if it ran locally.

```console
$ connex up            # on each machine, once
$ connex run -- ffmpeg -i big.mov out.mp4
connex → studio        # ran on the Mac Studio; output streamed back here
```

No IPs, no config files, no accounts, no server. A Thunderbolt cable, Ethernet,
USB4, and Wi-Fi all work the same way — connex discovers peers on every interface,
and the cable is just the fastest one.

## Why

Every desk with more than one computer wastes most of them. Wiring them together
today means SSH configs, IP addresses, and hostfiles. connex is the missing
fabric: find machines, know their capacity, put a process on one (or all) with
streamed I/O — and nothing more. Distributed inference launchers (MLX, llama.cpp
RPC, exo) run on top with one command instead of a page of setup.

## Install

```sh
# From source (Go 1.26+)
go install github.com/oddurs/connex/cmd/connex@latest

# Homebrew (coming with v0.2)
# brew install oddurs/tap/connex
```

## Quickstart

```sh
# 1. Start the agent on each machine (background)
connex up -d

# 2. Pair a second machine, one time
connex key                 # on machine A → prints 64 hex chars
connex join <key>          # on machine B

# 3. Use the fabric
connex ls                  # see machines and live load
connex run -- make -j      # runs where there's most headroom
connex run --on studio -- python quantize.py
cat data.csv | connex run -- zstd -19 > data.zst
connex each -- uname -a    # run everywhere at once
```

## Commands

| Command | Does |
| ------- | ---- |
| `connex up [-d]` | start sharing this machine's compute (`-d`: background) |
| `connex down` | stop the background agent |
| `connex ls [--json]` | list machines and live capacity |
| `connex run [--on NAME] -- CMD` | run where there's most headroom |
| `connex each -- CMD` | run on every machine at once |
| `connex key` | print this machine's cluster key |
| `connex join KEY` | trust the cluster that key belongs to |

## How it works

- **Discovery** — multicast-DNS on every interface (`_connex._tcp`), with a short
  client-side cache so back-to-back commands are instant.
- **Auth** — machines sharing a 32-byte key trust each other. Each connection is
  authenticated with an HMAC over a fresh server nonce; the key never crosses the
  wire. `run` is arbitrary code execution by design, so trust is the whole game.
- **Scheduling** — score each machine as load-per-core and pick the lowest; the
  local machine is always a candidate, and if it wins, the command runs locally
  and connex prints nothing.
- **Streaming** — one TCP connection per command, `TCP_NODELAY` on, output framed
  so it is binary-clean; the hot path allocates nothing per frame.

Full design: [`docs/`](docs/). The staged plan and its validator: [`roadmaps/`](roadmaps/).

## Security

Trusted local links only. v1 authenticates but does not yet encrypt the link, so
use it on a Thunderbolt cable or a home LAN, not a hostile network. Transport
encryption (TLS 1.3 with pinned certs) is planned for v0.4. See
[`docs/08-security-model.md`](docs/08-security-model.md) and [`SECURITY.md`](SECURITY.md).

## Status

v0.1 (the fabric) is in progress: `up`/`down`/`ls`/`run`/`each`/`key`/`join` work
end to end over loopback and LAN. Track progress in
[`roadmaps/STATUS.md`](roadmaps/STATUS.md).

## License

MIT — see [`LICENSE`](LICENSE).
