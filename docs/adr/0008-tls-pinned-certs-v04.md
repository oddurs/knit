# ADR-0008: TLS 1.3 with pinned self-signed certs (deferred to v0.4)
Status: Accepted (deferred)
Date: 2026-09-01

## Context
v1's link is plaintext: confidentiality, integrity against an active MITM, and
server authentication are all absent (see [ADR-0004](0004-hmac-nonce-auth.md) and
[08-security-model.md](../08-security-model.md)). Acceptable on a Thunderbolt cable
or trusted LAN, wrong on a hostile network — and a blocker for `connex proxy`, which
would carry sensitive tunneled traffic.

## Decision
Add TLS 1.3 in v0.4 without sacrificing zero-config: each machine generates a
self-signed certificate on first run; at `connex join` time the two machines
exchange and **pin each other's cert fingerprint** alongside the shared key.
Thereafter connections are TLS 1.3 with pinned mutual verification. No CA, no
certbot, no filenames for the user to manage. Deferred — not built in v1 — because
v1's job is the fabric on trusted links, and auth (which v1 has) is separable from
encryption (which v1 defers).

## Alternatives considered
- **Ship TLS in v1** — correct security sooner, but adds cert lifecycle to the very
  first release and isn't needed to gate execution on a trusted cable. Sequencing,
  not rejection.
- **Public CA / Let's Encrypt** — impossible for link-local/ephemeral hosts and
  drags in the internet; violates the local-only posture.
- **WireGuard/Tailscale as the encrypted transport** — great for cross-site
  (`--peer` already rides it), but requiring an overlay for two machines on one
  cable breaks zero-config for the primary use case.

## Consequences
- Closes the confidentiality, integrity, and server-auth gaps while staying
  zero-config; pinning happens inside the existing `join` flow.
- Costs one extra handshake RTT and negligible streaming overhead with hardware AES.
- Unblocks `connex proxy` ([CX-NET-040](../../roadmaps/milestones/m4-v0.4-encrypted-persistent.md)),
  which must not ship before it.
- The CONNEX1 application handshake is unchanged — TLS wraps the same bytes.
