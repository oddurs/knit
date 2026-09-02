# Vision

## The idea

Every desk with more than one computer is wasting most of them. A Mac Studio
idles while a MacBook Air's fans spin on a compile. Two machines with 128 GB of
unified memory each could hold a model neither fits alone — but wiring that up
today means SSH configs, IP addresses, hostfiles, and tutorials.

connex is the missing fabric: a tool so small and quiet it is almost invisible.

```
$ connex up          # on each machine, once
$ connex run -- ffmpeg -i big.mov out.mp4
connex → studio      # ran on the Mac Studio; output streamed back here
```

No IPs, no config files, no accounts, no server. One static binary.

## The three-word brief

**Simple. Fast. Seamless.** In that priority order.

- **Simple** is a constraint on us: one binary, one dependency, seven commands,
  a protocol that fits on a page. If a feature cannot be added without a config
  file or an eighth command, it waits or dies.
- **Fast** is a promise to the machine: the streaming path runs at the speed of
  the cable, and connex's own overhead on the hot path is measured in
  microseconds, not milliseconds. See the [performance budget](10-performance.md).
- **Seamless** is a promise to the person: you cannot feel that the work left.
  Pipes, exit codes, stdin EOF, and signals behave exactly as if the command ran
  under your shell. The only tell is one dim line on stderr — and only when the
  work actually left this machine.

## Principles

1. **Zero config is the feature.** Anything that requires editing a file or
   knowing an IP address is a bug. Discovery, port selection, and scheduling are
   automatic. The only deliberate act is trust (`connex join`), one command.

2. **Invisible by default.** `connex run` prints exactly one dim line to stderr
   saying where the command ran — and nothing at all when it ran locally. Exit
   codes, stdin, stdout, and stderr behave byte-for-byte as if the command were
   local, so connex composes with pipes:
   `cat data.csv | connex run -- zstd -19 > data.zst`.

3. **The cable is not a special case.** A Thunderbolt cable between two Macs, a
   USB4 link, an Ethernet run, and Wi-Fi all surface as IP interfaces on macOS
   and Linux — Thunderbolt Bridge appears automatically when you plug the cable
   in. connex does multicast-DNS discovery on *every* interface, so "share by
   cable" and "share by Wi-Fi" are the same code path. The cable is just the
   fastest one, and connex learns to prefer it (v0.3) without special-casing it.

4. **Do less, enable more.** connex does not shard tensors or implement RPC for
   ML frameworks. It provides the three primitives everything else needs: *find
   machines*, *know their capacity*, *put a process on one (or all) with streamed
   I/O*. MLX distributed, llama.cpp RPC, and exo run on top with one command
   instead of a page of setup (see [05-ai-workloads.md](05-ai-workloads.md)).

5. **A single binary you can read in an afternoon.** A small Go program: a
   handful of tightly-scoped internal packages, one runtime dependency (mDNS), a
   wire protocol on one page. Hypermodern in behavior, boring in construction.

6. **Fast is a feature, and it is designed in, not tuned in later.** The hot path
   has a latency budget ([10-performance.md](10-performance.md)) that every change
   is measured against: no per-frame heap allocation, one `writev` syscall per
   frame, Nagle disabled, zero-copy where the OS offers it, discovery cached so
   the second command in a row never stalls. Speed lives in the design, so the
   code stays simple.

## Why now

Unified-memory laptops and desktops are everywhere, Thunderbolt 4/5 gives a
40–120 Gbit/s sub-millisecond link for the price of a cable, and the AI tooling
that wants a small cluster (MLX, llama.cpp, exo, torchrun) all stall on the same
boring question: *which machines, at what addresses, and how do I start a process
on each?* connex answers exactly that and nothing more.

## What connex is not (v1)

- Not a container platform, not Kubernetes for the home. Commands run as the
  agent's user on the target machine.
- Not a file-sync tool — v1 streams stdio only; commands needing input files
  should read from stdin or from paths that exist on the target. Directory sync
  is roadmap ([`CX-EXEC-030`](../roadmaps/milestones/m2-v0.2-real-use.md)).
- Not internet-facing. Trusted local links only; see
  [08-security-model.md](08-security-model.md).
- Not a daemon zoo. The agent is the only long-lived process; the client is
  fresh per command and keeps no background state beyond a short discovery cache.
