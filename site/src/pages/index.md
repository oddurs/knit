---
layout: ../layouts/Base.astro
---
# Your machines, one pool of compute

knit lets the computers around you share their hardware. Run a command and
it executes on whichever machine has the most headroom, and everything comes
back as if it had run right here. No configuration, no accounts, no
addresses. One small binary.

```
$ knit run -- ffmpeg -i big.mov out.mp4
knit → studio
```

That is the whole product. The laptop was busy, the Mac Studio on the desk was
not, so the transcode ran there. Its output streamed back, its exit code came
with it, and the one dim line is the only sign the work left the room. Had the
laptop been the better choice, knit would have printed nothing at all.

## Nothing to configure

Machines find each other on their own, on every interface they have. Plug a
Thunderbolt cable between two Macs and they are peers a second later; the same
happens over Ethernet and Wi-Fi. Pairing is one key, pasted once. There is no
config file, no daemon to describe, no list of hosts to keep current.

## As fast as the cable

A Thunderbolt 4 cable is a forty-gigabit link with sub-millisecond latency.
knit streams over it at line rate: one write per chunk, nothing allocated on
the way, nothing buffered. The machine on the other end is the limit, not
knit.

## Exactly as if it ran here

stdin, stdout, stderr, pipes, exit codes, Ctrl-C. All of it behaves
byte-for-byte as a local command would, so scripts, `set -e`, and `| wc -l`
work unchanged. If the connection drops, the remote process is stopped; nothing
is ever left running on the other machine.

## What it is used for

- Offloading the heavy thing: a transcode, a build, a quantization, an eval.
- Pipes of any size: `cat 50GB.log | knit run -- zstd -19 > out.zst`.
- One command on every machine: `knit each -- brew upgrade`.
- Small AI clusters: `knit run --mem 48` puts a model where it fits, and
  `knit each` starts a distributed job with the ranks and hosts already set.

## Two minutes to the first run

```sh
brew install oddurs/tap/knit
knit up -d
```

On the second machine, `knit join` the key the first one prints, and you are
done. [Two machines in two minutes](/knit/docs/getting-started/quickstart/)
walks through it; [Commands](/knit/docs/reference/commands/) is the whole
surface, seven of them.
