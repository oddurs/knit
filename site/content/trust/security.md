---
title: Security
order: 1
---
# Security

Short version: every connection is encrypted, both ends prove they hold the
key, and holding the key is a shell on every machine in the fabric. Guard the
key; the network no longer has to be trusted.

## What the key protects

Every connection is TLS 1.3. Inside it, the agent sends a fresh random nonce
and the client answers with an HMAC of that nonce under the shared key, bound
to this particular connection; the agent then proves the key back the same
way before the client sends anything. The key never crosses the wire, a
recorded answer is useless because the nonce is single-use, and an answer
relayed through a machine in the middle fails because it was bound to a
different connection. Comparison is constant-time.

So: anyone who does not hold the key can list nothing, run nothing, read
nothing that passes, and learn only that a knit agent exists. An agent that
does not hold your key is refused too, so a stale or impostor machine never
receives a command.

## What the key grants

Anyone who does hold the key can run any command on your machine, as the user
the agent runs as, with that user's files and credentials. That is the point,
and it is the whole threat model: **holding the key is the same as having a
shell on every machine in the fabric.** Treat `~/.knit/key` like an SSH private
key, and only `knit join` a key from a machine you would let log in to yours.

## Revoking a machine

There is no per-machine identity, so revocation is rotation:

```sh
knit key --rotate
```

installs a fresh key on this machine, prints it, and names the machines that
were reachable under the old one. Run `knit join <key>` on the ones you keep;
the rest are out. Agents pick the new key up on their next connection, with no
restart.

## Networks

A Thunderbolt cable, an office LAN, a hotel Wi-Fi: the link does not need to
be trusted. What still applies:

- Do not forward the agent's port through a router. There is nothing to gain,
  and an exposed agent is an invitation to guess at your key.
- Across sites, use [Tailscale or another overlay](../guides/peers.md) with
  `--peer`. knit rides inside it; the overlay adds reachability and its own
  identity, not a substitute for the key.

## What the agent exposes

One TCP port (5648 when free), the machine's name, and, to key holders, its
core count, memory, load, and accelerator. The agent runs commands in its own
user's home directory and stops them when the client goes away. It does not
open anything else and has no configuration to get wrong.

## Reporting

Security issues: see [SECURITY.md](https://github.com/oddurs/knit/blob/main/SECURITY.md)
in the repository.
