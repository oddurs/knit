---
title: Machines discovery can't see
order: 6
---
# Machines discovery can't see

Automatic discovery uses multicast, which does not cross routers and is
filtered on some office and guest networks. For those machines, and for peers
on a Tailscale tailnet or any other overlay, name them.

## --peer

```sh
knit gauge --peer studio.tail1234.ts.net
knit run --peer 100.101.102.103 -- make
```

`--peer` works on `gauge`, `run`, and `each`, may be repeated, and goes before
`--`. The named machine is probed like any discovered one: it still needs the
key, and it is still scored on load. Discovered and named machines are merged,
so adding a peer never hides one found automatically.

## KNIT_PEERS

For machines you always want, put them in the environment:

```sh
set -x KNIT_PEERS studio.tail1234.ts.net,mini.tail1234.ts.net   # fish
export KNIT_PEERS=studio.tail1234.ts.net,mini.tail1234.ts.net   # bash, zsh
```

Comma-separated, same forms as `--peer`.

## Ports

An agent listens on port 5648 whenever that port is free, so a bare hostname is
enough. If 5648 was taken, the agent picks a random port instead; discovery still
finds it, but a `--peer` entry then needs `host:port`. The agent's log
(`~/.knit/agent.log`) says which port it bound.

If a firewall sits between the machines, allow TCP 5648 inbound on the agent
side.

## Tailscale

This is the intended way to use knit across the internet. Tailscale gives every
machine a stable name and an encrypted link; knit rides on top:

```sh
knit run --peer studio.tail1234.ts.net --on studio -- ...
```

knit's own link is not encrypted yet (see [Security](../trust/security.md)),
which is exactly why an overlay is the right transport for anything beyond
your desk or LAN.
