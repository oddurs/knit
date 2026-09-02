# Security policy

## Trust model

connex's `run` operation is **arbitrary code execution on the target machine, by
design.** Machines that hold the same 32-byte cluster key trust each other
completely. Holding the key is authorization to run anything as the agent's user.
Authentication is therefore the entire security boundary. The full model, the
accepted v1 gaps, and the TLS plan are in
[`docs/08-security-model.md`](docs/08-security-model.md).

## What v1 defends, and what it does not

- **Defends:** replay (per-connection nonce), passive key theft (HMAC, key never
  on the wire), and unauthenticated execution.
- **Does not defend (v1):** confidentiality and active man-in-the-middle — the
  link is plaintext. Use connex on a Thunderbolt cable or a trusted LAN only.
  Encrypted transport (TLS 1.3 with pinned self-signed certs) is planned for v0.4.

Never expose the agent port beyond local links. For use across sites, put the
machines on a Tailscale tailnet and use `--peer host:port` (v0.2).

## Reporting a vulnerability

Please report suspected vulnerabilities privately by opening a
[GitHub security advisory](https://github.com/oddurs/connex/security/advisories/new)
or emailing the maintainer, rather than filing a public issue. Describe the class
of problem and how to reproduce it; please do not include a working exploit in a
public channel. Expect an initial response within a few days.
