# Security model

knit's `run` op is **arbitrary code execution on the target machine, by
design.** There is no sandbox, no allowed-command list, no per-command
authorization. That is a deliberate choice for a tool whose job is to run your
commands on your other machines. It means authentication is not one feature among
many — it is the entire security boundary. This document is honest about what that
boundary defends and what, in v1, it does not.

## Trust model

**Machines that hold the same 32-byte key trust each other completely.** Holding
the key is authorization to run anything as the agent's user. There are no roles,
scopes, or quotas. Pairing is the only deliberate trust act, and it is one
command each way:

```
knit key            # machine A prints 64 hex chars (the key)
knit join <key>     # machine B installs it — now A and B are one cluster
```

- The key lives at `~/.knit/key`, mode `0600`, auto-generated with
  `crypto/rand` on first use. `KNIT_HOME` relocates it.
- To revoke a machine, rotate the key: generate a new one and re-`join` the
  machines you still trust. (Key rotation UX is a v0.4 nicety,
  [`KN-AUTH-041`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md); the
  mechanism — overwrite the keyfile — works today.)

## Authentication: HMAC over a per-connection nonce

- The agent sends a fresh 16-byte random nonce per connection.
- The client replies with `HMAC-SHA256(key, nonce)`.
- The agent recomputes and compares in constant time (`hmac.Equal`).

Properties this gives us, and their limits:

| Property                                   | v1? | Note |
| ------------------------------------------ | --- | ---- |
| Key never crosses the wire                 | ✓   | only the HMAC does |
| Replay of a captured HMAC is useless       | ✓   | nonce is single-use, random |
| A passive listener learns nothing usable   | ✓   | HMAC over a nonce it can't reuse |
| Mutual authentication                      | ✗   | the *client* proves knowledge to the server; the client does not verify the server's identity (v1) |
| Confidentiality of command text and output | ✗   | plaintext on the wire (v1) |
| Integrity against an active MITM           | ✗   | an active attacker on-path could hijack the authenticated stream (v1) |

## Accepted v1 gaps (documented, not hidden)

1. **No transport encryption.** Command text and output cross the link in the
   clear. On a Thunderbolt cable (a physical point-to-point wire) or a trusted
   home LAN this is acceptable. On a hostile or shared network it is not. knit
   prints no false assurance; the docs say plainly: trusted local links only.

2. **No server authentication / MITM resistance.** Because only the client
   authenticates, an active attacker who can intercept and modify traffic on the
   path could impersonate an agent or tamper with a stream. Same mitigation:
   trusted links only, until TLS lands.

3. **No per-command authorization or quotas.** Trust is all-or-nothing.
   Deliberate — quotas are where invisible tools go to die. Revisit only if real
   usage demands it, and never at the cost of the zero-config posture.

## The fix: TLS 1.3 with pinned self-signed certs (v0.4)

Planned in [`KN-AUTH-040`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md).
Each machine generates a self-signed certificate on first run. At `join` time the
two machines exchange and pin each other's cert fingerprint (alongside the shared
key). Thereafter connections run TLS 1.3:

- **Confidentiality + integrity** against passive and active attackers on the
  path — closes gaps 1 and 2.
- **Mutual authentication** via pinned fingerprints — each end verifies the other
  is a machine it explicitly paired with, not just "someone with the key."
- **Still zero-config** — no CA, no certbot, no filenames to manage; pinning
  happens inside the existing `join` flow.

TLS 1.3 costs one extra handshake round-trip and near-zero streaming overhead with
hardware AES. Only after TLS ships does the network-sharing `knit proxy`
([`KN-NET-040`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md)) become
acceptable, because a SOCKS tunnel carries traffic too sensitive for a plaintext
link.

## Network exposure

- The agent binds an **ephemeral port** and advertises it only over link-local
  mDNS. It is not, and will never be, an internet-facing service.
- knit will **not** implement internet rendezvous or hole-punching. Cross-site
  use rides an existing overlay: put both machines on a Tailscale tailnet and use
  `--peer host:port` ([`KN-DISC-020`](../roadmaps/milestones/m2-v0.2-real-use.md)).
  That is the supported "beyond the LAN" story, and it inherits the tailnet's
  encryption and identity.
- Never expose the agent port through a router/firewall. The docs and `knit up`
  banner say so.

## Denial of service

Out of scope for the threat model — a trusted peer that floods you is a trust
problem, not an auth problem, and the answer is to stop trusting it (rotate the
key). knit does cap frame size (1 MiB) and probe timeouts (250 ms) so a
malformed or slow peer cannot wedge a client, but it does not rate-limit a
correctly-authenticated peer.

## Reporting

Pre-1.0, security issues go to the maintainer privately (see repo root
`SECURITY.md`, [`KN-DOC-001`](../roadmaps/milestones/m1-v0.1-fabric.md)). No bounty
program at this stage; honest disclosure and a fast fix.
