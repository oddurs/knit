# ADR-0004: HMAC-SHA256 over a per-connection nonce
Status: Accepted
Date: 2026-09-01

## Context
`run` is arbitrary code execution by design, so authentication is the entire
security boundary. It must be zero-config (a single shared secret, no CA, no
accounts), safe against replay and passive listeners on a LAN, and implementable in
the stdlib with no dependency.

## Decision
Machines share a 32-byte key (`~/.connex/key`, 0600, `crypto/rand`-generated).
Per connection the agent sends a fresh 16-byte random nonce; the client returns
`HMAC-SHA256(key, nonce)`; the agent verifies in constant time (`hmac.Equal`). The
key never crosses the wire. Pairing is `connex key` / `connex join <key>`.

## Alternatives considered
- **Send the key / a bearer token directly** — trivially replayable and exposed to
  any passive listener. Rejected.
- **TLS mutual auth with certs, now** — the right long-term answer for
  confidentiality + MITM resistance, but heavier and not needed to gate execution
  on a trusted link. Deferred to [ADR-0008](0008-tls-pinned-certs-v04.md); auth and
  encryption are separable, and v1 needs auth.
- **Public-key (Ed25519) pairing** — stronger identity, but more moving parts than
  a shared symmetric secret for a "these machines are mine" trust model. Overkill
  for v1.

## Consequences
- Zero-config auth with replay resistance and no dependency.
- Known limits, documented honestly in [08-security-model.md](../08-security-model.md):
  the client authenticates to the server but not vice-versa, and the channel is
  plaintext — an active MITM is not defended in v1. TLS (v0.4) closes both without
  changing this handshake.
- Revocation is key rotation; fine-grained revocation is explicitly out of scope.
