# ADR-0009: TLS 1.3 with a key-bound handshake, not pinned certificates
Status: Accepted (supersedes the mechanism in ADR-0008)
Date: 2026-09-02

## Context
[ADR-0008](0008-tls-pinned-certs-v04.md) decided on TLS 1.3 for v0.4 and
sketched the trust mechanism as self-signed certificates whose fingerprints
are exchanged and pinned at `knit join`. Building it exposed two problems.
`join` is offline — a user pastes 64 hex characters — so there is no channel
in which A and B could exchange fingerprints; pinning would need a network
step or trust-on-first-use, either way new state per machine that breaks on
every reinstall. And the machines already share a secret, which is a
stronger basis for authenticating a channel than a certificate nobody
verifies.

## Decision
Every connection is TLS 1.3. The agent presents an ephemeral self-signed
certificate generated at start (in memory, never written); the client does
not verify it. Authentication is the existing HMAC exchange, now bound to
the connection: both sides derive 32 bytes of channel binding from the TLS
keying material (`ExportKeyingMaterial`), the client proves
`HMAC(key, "knit client" ‖ nonce ‖ binding)`, and the server answers with
`HMAC(key, "knit server" ‖ nonce ‖ binding)`, which the client checks before
trusting a single byte.

A machine in the middle terminating TLS on both legs sees a different
binding on each, so neither proof it forwards verifies. An agent that does
not hold the key cannot produce the server proof, so the client refuses it.
Nothing new is stored, nothing new is configured, and `join` is unchanged.

## Alternatives considered
- **Pinned certificates (ADR-0008 as written)** — needs a fingerprint
  exchange `join` has no channel for, plus cert state that reinstalls and
  rotations invalidate. The security it adds over a key-bound channel is
  per-machine identity, which `knit key --rotate` covers at the fabric level.
- **TLS 1.3 external PSK** — the ideal shape, but Go's `crypto/tls` does not
  expose external pre-shared keys. The exporter binding achieves the same
  property on top of a certificate handshake.
- **Trust-on-first-use pinning** — equivalent security to the key alone
  (the key gates the first use), with the state-management cost and none of
  the benefit.

## Consequences
- Confidentiality, integrity, and mutual authentication on every link, with
  zero new steps: the accepted gaps in [08-security-model.md](../08-security-model.md)
  close.
- Protocol version 3. A v3 client names a plaintext agent as "an older knit"
  and says to upgrade it; a plaintext client's probe of a TLS agent simply
  fails, so an un-upgraded machine drops out of `gauge` until it is upgraded.
- One extra round trip per connection and AES-GCM on the stream; measured
  at roughly 850 MB/s on loopback and to a VM, against 1.3 GB/s plaintext.
- Frames are written as one buffer (header and payload copied together) so
  each frame is one TLS record sequence rather than two.
- The agent re-reads the key per connection, so rotation needs no restart.
