---
layout: ../layouts/Base.astro
---
# knit

Share hardware with the machines around you. One binary, no configuration,
and commands that behave exactly as if they ran here.

## 1. Start

On each machine, once.

```
$ brew install oddurs/tap/knit
$ knit up -d
knit up (pid 4242) — this machine is now a loop in the fabric
```

## 2. Pair

One key, pasted once. That is the whole setup.

```
$ knit key                    # on the first machine
7c1e5b…9f04

$ knit join 7c1e5b…9f04       # on the second
knit: cluster key installed — this machine now trusts that cluster
```

## 3. Run

Work goes to the machine with the most headroom. Output comes back as if it
were local.

```
$ knit gauge
NAME     ADDR          OS/ARCH       CPUS  MEM     FREE   LOAD  GPU           LINK
studio   169.254.87.3  darwin/arm64  24    128.0G  96.4G  0.31  Apple M3 Max  thunderbolt ~40G
laptop   —             darwin/arm64  8     24.0G   9.8G   4.02  Apple M2      (this machine)

$ knit run -- ffmpeg -i big.mov out.mp4
knit → studio

$ cat corpus.jsonl | knit run -- python embed.py > vectors.jsonl
knit → studio

$ knit run -- sh -c 'exit 7'; echo $?
knit → studio
7

$ knit each -- uname -sr
[laptop] Darwin 25.5.0
[studio] Darwin 25.5.0
```

## What happened

knit found the other machine by itself, over the cable. The key proved they
trust each other. Each command ran where load per core was lowest, its output
streamed back, and its exit code came with it. The dim `knit → studio` line is
the only sign the work left the room; when this machine wins, knit prints
nothing at all.

Thunderbolt cable, Ethernet, Wi-Fi, or a Tailscale tailnet. macOS and Linux.
Free memory and accelerators are reported, so `knit run --mem 48` puts a model
where it fits and `knit each` starts distributed jobs with no hostfile.

[Two machines in two minutes](/knit/docs/getting-started/quickstart/) ·
[Commands](/knit/docs/reference/commands/) ·
[AI workloads](/knit/docs/guides/ai-workloads/) ·
[GitHub](https://github.com/oddurs/knit)
