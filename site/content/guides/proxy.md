---
title: Route through another machine
order: 9
---
# Route through another machine

```sh
knit proxy
```

opens a SOCKS5 proxy on `127.0.0.1:1080` that sends every connection through a
peer's agent. Point a program at it and that program reaches whatever network
the peer can: a laptop on hotel Wi-Fi routes through the wired desktop at home,
a Thunderbolt-only machine reaches the internet through its neighbor.

## Using it

Leave `knit proxy` running in one terminal, and point a client at the proxy:

```sh
curl --socks5-hostname 127.0.0.1:1080 https://example.com
```

Most browsers and tools take a SOCKS5 proxy the same way. `--socks5-hostname`
(rather than `--socks5`) resolves names on the peer, which is usually what you
want — the peer looks up and reaches the address.

## Choosing the peer

With one other machine reachable, `knit proxy` uses it. With several, name one:

```sh
knit proxy --on desktop
```

Change the local port with `--port`:

```sh
knit proxy --port 8080
```

## What it is and isn't

Each tunneled connection is a TLS-encrypted, key-authenticated `dial` through
the peer's agent — the same transport as `knit run`. The proxy listens on
`127.0.0.1` only, so nothing off your machine can use it.

It carries traffic to wherever the peer can reach, including the peer's own
`localhost`. Anyone holding the key can already run commands on the peer, so
tunneling through it grants nothing new. It is not an anonymity tool and not a
VPN for the whole system; it is a per-application route through a machine you
already trust.
