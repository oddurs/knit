# Security policy

## Trust model

knit's `run` operation is **arbitrary code execution on the target machine, by
design.** Machines that hold the same 32-byte cluster key trust each other
completely. Holding the key is authorization to run anything as the agent's user.
Authentication is therefore the entire security boundary. The full model and
its accepted gaps are in [`docs/08-security-model.md`](docs/08-security-model.md).

## What knit defends, and what it does not

- **Defends:** confidentiality and integrity of every connection (TLS 1.3),
  mutual authentication bound to the connection (an agent or client without
  the key is refused, and a machine in the middle cannot forward a proof),
  replay (per-connection nonce), and passive key theft (the key never crosses
  the wire).
- **Does not defend:** anything a legitimate key holder does. Holding the key is
  a shell on every machine in the fabric; there is no per-machine identity and
  no per-command authorization. Rotate the key (`knit key --rotate`) to revoke.

Do not expose the agent port through a router; there is nothing to gain from it.
For use across sites, put the machines on a Tailscale tailnet and use
`--peer host`.

## Reporting a vulnerability

Please report suspected vulnerabilities privately by opening a
[GitHub security advisory](https://github.com/oddurs/knit/security/advisories/new)
or emailing the maintainer, rather than filing a public issue. Describe the class
of problem and how to reproduce it; please do not include a working exploit in a
public channel. Expect an initial response within a few days.
