---
title: Two machines in two minutes
order: 2
---
# Two machines in two minutes

You need two machines with knit [installed](install.md), on the same network,
or joined by a cable. Call them A and B.

## 1. Start the agent on both

```sh
knit up -d
```

Each machine is now advertising itself and its spare capacity. `-d` keeps the
agent running in the background; `knit down` stops it.

## 2. Pair them, once

On A:

```sh
knit key
```

It prints 64 hex characters: the cluster key. On B:

```sh
knit join <key>
```

That is the only setup there is. Any machine holding the key is in the fabric;
any machine without it is refused.

## 3. See the fabric

```sh
knit gauge
```

```
NAME     ADDR            OS/ARCH        CPUS  MEM     FREE   LOAD  GPU           LINK
studio   169.254.87.3    darwin/arm64   24    128.0G  96.4G  0.31  Apple M3 Max  thunderbolt ~40G
here     —               darwin/arm64   8     24.0G   9.8G   4.02  Apple M2      (this machine)
```

Both machines listed, with live load and the link between them. `knit ls` is an
alias if your fingers prefer it.

## 4. Run something

```sh
echo hello | knit run -- tr a-z A-Z
```

```
knit → studio
HELLO
```

The dim line on stderr says where the work went. If this machine had the most
headroom, the command runs here and knit prints nothing at all.

Pin a machine with `--on`:

```sh
knit run --on studio -- uname -a
```

And that is knit. Everything else is detail: [what travels with a
command](../guides/files.md), [how a machine is chosen](../guides/placement.md),
and [running on every machine at once](../guides/each.md).
