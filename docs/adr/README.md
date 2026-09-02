# Architecture Decision Records

Each ADR captures one decision that is expensive to reverse: the context, the
choice, the alternatives weighed, and the consequences accepted. They are
immutable once `Accepted` — a changed mind gets a new ADR that supersedes the old
one, so the reasoning trail stays intact.

| ADR | Decision | Status |
| --- | -------- | ------ |
| [0001](0001-language-go.md) | Go as the implementation language | Accepted |
| [0002](0002-internal-package-layout.md) | Small internal packages, not one flat `main` | Accepted (supersedes the vision's "flat package") |
| [0003](0003-mdns-discovery-and-cache.md) | mDNS discovery + short client-side peer cache | Accepted |
| [0004](0004-hmac-nonce-auth.md) | HMAC-SHA256 over a per-connection nonce for auth | Accepted |
| [0005](0005-tcp-length-prefixed-framing.md) | Raw TCP + length-prefixed frames, one op per connection | Accepted |
| [0006](0006-load-per-core-scheduler.md) | Load-per-core scheduling, local always a candidate | Accepted |
| [0007](0007-single-static-binary.md) | One static CGo-free binary, both roles | Accepted |
| [0008](0008-tls-pinned-certs-v04.md) | TLS 1.3 with pinned self-signed certs (deferred to v0.4) | Accepted (deferred) |

## Template

```
# ADR-NNNN: <title>
Status: Proposed | Accepted | Superseded by ADR-XXXX
Date: YYYY-MM-DD

## Context
What forces are at play — technical, product, the north star.

## Decision
The choice, stated plainly.

## Alternatives considered
Each with the reason it lost.

## Consequences
What gets easier, what gets harder, what we now have to live with.
```
