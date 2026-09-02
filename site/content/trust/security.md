---
title: Security
order: 1
---
# Security

Short version: the key decides who may run commands on your machine, and the
connection is not encrypted yet. Use knit on a cable, a LAN you trust, or an
overlay such as Tailscale.

## What the key protects

Every connection starts with the agent sending a fresh random nonce and the
client answering with an HMAC-SHA256 of it under the shared key. The key never
crosses the wire, a recorded answer is useless because the nonce is single-use,
and comparison is constant-time. Without the key an agent answers exactly one
line, `unauthorized`, and closes.

So: anyone who does not hold the key can list nothing, run nothing, and learn
only that a knit agent exists.

## What the key grants

Anyone who does hold the key can run any command on your machine, as the user
the agent runs as, with that user's files and credentials. That is the point,
and it is the whole threat model: **holding the key is the same as having a
shell on every machine in the fabric.** Treat `~/.knit/key` like an SSH private
key, and only `knit join` a key from a machine you would let log in to yours.

To revoke a machine, generate a new key (`rm ~/.knit/key && knit key`) and
`knit join` it on the machines you keep. There is no per-machine identity to
revoke individually.

## What is not protected

The connection is plaintext TCP. On a network you do not control, someone in
the path can read the commands you send and the output that comes back, and
could alter them in flight. Authentication stops them from starting their own
commands; it does not hide yours.

Transport encryption with pinned certificates is planned for v0.4. Until then
the guidance is simple:

- A Thunderbolt cable, an Ethernet cable, or your own home or office LAN: fine.
- Anything further, or any network shared with people you do not trust: put
  knit on Tailscale (or another overlay) and use [`--peer`](../guides/peers.md).
  The overlay encrypts and authenticates the link; knit rides inside it.

## What the agent exposes

One TCP port (5648 when free), the machine's name, and, to key holders, its
core count, memory, load, and accelerator. The agent runs commands in its own
user's home directory and stops them when the client goes away. It does not
open anything else and has no configuration to get wrong.

## Reporting

Security issues: see [SECURITY.md](https://github.com/oddurs/knit/blob/main/SECURITY.md)
in the repository.
