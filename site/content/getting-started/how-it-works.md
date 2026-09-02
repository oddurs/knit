---
title: How it works
order: 3
---
# How it works

Five things to hold in your head. Nothing else is required.

## One binary, two roles

The same `knit` binary is the agent (`knit up`) and the client (everything
else). An agent listens for work; a client finds agents and sends work to them.
Every machine normally runs both: it offers its capacity and uses everyone
else's.

## Machines find each other by themselves

An agent announces itself over multicast DNS, the same mechanism that makes
printers appear on a network, on every interface the machine has. A client
browses every interface too. A Thunderbolt cable between two Macs becomes a
network interface the moment it is plugged in, so a cable and a Wi-Fi network
are found the same way, with no addresses typed anywhere.

Networks that block multicast, and overlays such as Tailscale, are handled by
[naming the machine explicitly](../guides/peers.md).

## One key is the whole pairing

Every machine in a fabric holds the same 32-byte key in `~/.knit/key`. A client
proves it has the key on every connection without ever sending it. That is
authentication in full: no accounts, no certificates, no allowlists. Treat the
key like an SSH private key; [Security](../trust/security.md) says exactly what
it does and does not protect.

## The least-loaded machine wins

Before every `knit run`, the client asks each reachable agent for its live load,
free memory, and accelerator, in parallel, within a quarter of a second. It
scores each machine, including itself, as load per core and picks the lowest.
[Choose the right machine](../guides/placement.md) covers how to constrain that.

## Streams, not copies

A remote command's stdin, stdout, and stderr are streamed over one TCP
connection as they happen, so pipes work at any size and interactive tools see
input as you type it. The exit code comes back last and becomes knit's own exit
code. Ctrl-C is forwarded as a real signal, and if the connection drops for any
reason the agent stops the remote process rather than leaving it running.
