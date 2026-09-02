# Security model

knit's `run` op is **arbitrary code execution on the target machine, by
design.** There is no sandbox, no allowed-command list, no per-command
authorization. That is a deliberate choice for a tool whose job is to run your
commands on your other machines. It means authentication is not one feature among
many — it is the entire security boundary. This document says what that boundary
defends and what it deliberately does not.

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
- To revoke a machine, rotate the key: `knit key --rotate` installs a fresh key
  atomically, prints it, and names the machines that were reachable under the
  old one so you know where to `knit join`
  ([`KN-AUTH-041`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md)).
  The agent re-reads the key on every connection, so rotation needs no restart.

## Transport: TLS 1.3, authenticated by the key

Every connection is TLS 1.3 ([ADR-0009](adr/0009-tls-key-bound-handshake.md)).
The agent presents an ephemeral self-signed certificate that clients do not
verify; the certificate only keys the channel. What authenticates both ends is
the shared key, bound to that specific connection:

1. Both sides derive 32 bytes of *channel binding* from the TLS keying
   material. It is unique to this connection; a relayed connection has a
   different one on each leg.
2. The agent sends a fresh 16-byte random nonce.
3. The client replies with `HMAC-SHA256(key, "knit client" ‖ nonce ‖ binding)`.
4. The agent verifies in constant time, then answers with
   `HMAC-SHA256(key, "knit server" ‖ nonce ‖ binding)`, which the client
   verifies before it sends a command or believes a capacity report.

| Property                                   | Note |
| ------------------------------------------ | ---- |
| Key never crosses the wire                 | only proofs do |
| Replay of a captured proof is useless      | nonce is single-use; binding is per connection |
| A passive listener learns nothing          | TLS; command text and output are encrypted |
| Integrity against an active attacker       | TLS; and a proof forwarded across a relay fails the binding check |
| Mutual authentication                      | the client proves the key to the agent, the agent proves it back |
| An agent with a different key is refused   | its server proof does not verify; the client says so |

## Accepted gaps (documented, not hidden)

1. **No per-machine identity.** Everyone with the key is equally trusted; you
   cannot revoke one machine without rotating for all. Deliberate — per-machine
   certificates would add state that breaks on reinstall, for a fabric that is
   usually two or three machines on a desk.

2. **No per-command authorization or quotas.** Trust is all-or-nothing.
   Deliberate — quotas are where invisible tools go to die. Revisit only if real
   usage demands it, and never at the cost of the zero-config posture.

3. **The agent runs commands as its user.** Anyone holding the key has that
   user's files and credentials on that machine. Treat the key as you would an
   SSH private key.

## Network exposure

- The agent binds port 5648 when free (else an ephemeral port) and advertises
  it over link-local mDNS. It is not an internet-facing service and knit will
  not implement internet rendezvous or hole-punching. Cross-site use rides an
  existing overlay: put both machines on a Tailscale tailnet and use
  `--peer host` ([`KN-DISC-020`](../roadmaps/milestones/m2-v0.2-real-use.md)).
- With TLS the link no longer needs to be trusted, but the port still should
  not be exposed through a router: an exposed agent is an oracle for anyone
  who wants to guess at your key, and there is nothing to gain from it.

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
