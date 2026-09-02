---
title: Cables, Ethernet, Wi-Fi
order: 5
---
# Cables, Ethernet, Wi-Fi

knit works over any IP link and treats them all the same way. It only differs
in what it tells you and, on ties, which one it prefers.

## Thunderbolt

Plug a Thunderbolt or USB4 cable between two Macs. macOS creates a
"Thunderbolt Bridge" interface on both, gives it an address, and knit finds the
other machine over it within a second, with nothing configured. Linux does the
equivalent with a `thunderbolt0` interface.

`knit gauge` shows the link as `thunderbolt ~40G`. Real throughput on a
Thunderbolt 4 bridge is well past what most commands can consume; the machine on
the other end, not the cable, is the limit. This is the link for anything that
streams a lot of data: transcodes, large stdin pipes, pipeline-parallel model
inference.

## Ethernet

Shown as `ethernet` with the negotiated speed, for example `ethernet 10G`,
when the OS reports it. Ordinary gigabit Ethernet moves about 110 MB/s, enough
for most commands, slow for shipping large model layers between machines every
token.

## Wi-Fi

Shown as `wifi`. Fine for launching commands, checking on machines, and small
data. Honest note: any workload that moves activations or weights between
machines continuously will disappoint over Wi-Fi. Use the cable.

## Several links at once

A machine reachable both over a cable and over Wi-Fi is dialed over the cable.
knit keeps the fastest address it discovers for each peer, and
[placement](placement.md) breaks ties toward the faster link.

## `lan` and `net`

`lan` means the peer shares a subnet with an interface the OS could not
describe, typically a virtual bridge or a VM. `net` means the peer is reached
through a router or an overlay such as Tailscale; speed is unknown and knit
does not guess.
