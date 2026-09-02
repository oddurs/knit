# AI workloads

The reason two idle Macs on a desk matter in 2026: unified memory. Two machines
with 64 GB each can serve a model neither fits alone, and a Thunderbolt 4/5 cable
(40–120 Gbit/s, sub-millisecond latency) is a genuinely usable interconnect for
pipeline-parallel inference. The tooling exists — MLX distributed, llama.cpp RPC,
exo — but all of it stalls on the same boring problem knit solves: *which
machines, what addresses, how do I start a process on each of them?*

## What knit contributes

knit stays a fabric, not an ML runtime (principle 4). It gives AI tooling three
primitives, and the speed to make them worth using:

1. **Capacity-aware discovery** — `knit ls --json` returns every machine with
   arch, cores, total and free memory, and accelerator. A launcher answers "can
   this cluster fit a 70B q4 model, and how should layers split?" without a config
   file.
2. **Fan-out process launch** — `knit each` starts a worker on every machine
   with streamed logs and propagated failure codes.
3. **Placed execution** — `knit run --on` / `--mem` puts a heavy job on the
   machine that can actually hold it.

## Why the transport speed matters here specifically

Pipeline-parallel inference ships activations between stages every token. knit's
framing overhead (one `writev` per frame, pooled buffers, Nagle off) keeps the
per-hop cost near the wire, so the cable — not knit — is the ceiling. The honest
physics, repeated wherever it belongs: **tensor-parallel across machines is
bandwidth-hungry and disappointing over Wi-Fi; pipeline/layer splits over a
Thunderbolt cable are the sweet spot.** knit surfaces link type per peer (v0.3
`ls` shows `thunderbolt`/`wifi`) so users aren't surprised, and prefers the wired
address when a peer is reachable on several.

## Concrete flows

**Split inference with llama.cpp RPC** (works with v0.1 primitives):

```sh
knit each -- ./rpc-server -H 0.0.0.0 -p 50052       # workers everywhere
llama-cli -m 70b-q4.gguf --rpc studio.local:50052,mini.local:50052 -ngl 99
```

**MLX distributed** wants a hostfile; generating it is one `ls --json` away.
Planned sugar (v0.3, [`KN-AI-030`](../roadmaps/milestones/m3-v0.3-ai-native.md)):
`knit hostfile > hosts.json` and `knit mpirun -- python -m mlx_lm.generate ...`
so the whole flow is two commands. The same shape drives `torchrun --nnodes` on
Linux boxes.

**Quantization / batch offload** (works with v0.1): quantizing, dataset
preprocessing, and evals are exactly the "one heavy command, big I/O" shape knit
streams well:

```sh
knit run --on studio -- python -m mlx_lm.convert --hf-path meta-llama/... -q
cat corpus.jsonl | knit run -- python embed.py > vectors.jsonl
```

## Memory and accelerator reporting (v0.3)

Total RAM ships in v0.1 `info`. To schedule models properly, `info` grows
([`KN-SYS-030`](../roadmaps/milestones/m3-v0.3-ai-native.md)):

- `mem_free_gb` — what could actually be allocated now, so `--mem` is meaningful.
- `gpu` — on Apple silicon, chip name + unified memory (one `sysctl` /
  `system_profiler` away); on Linux, NVML totals if present.
- `accel` — `metal` | `cuda` | `none`, so launchers pick backends.

## Sharing network resources (v0.4 sketch)

Same fabric, different resource: a machine's *connectivity* is just another thing
an agent can serve ([`KN-NET-040`](../roadmaps/milestones/m4-v0.4-encrypted-persistent.md)).

- `knit proxy [--on NAME]` — a local SOCKS5 listener that tunnels through the
  chosen peer's agent connection. Laptop on hotel Wi-Fi uses the wired machine's
  network; a Thunderbolt-only machine reaches the world through its peer.
- Implementation is small: one new op (`dial host:port`) reusing the exact auth +
  streaming machinery from `run`. **This must wait for transport encryption (TLS,
  v0.4)** — proxied traffic is too sensitive for the v1 plaintext link, and
  shipping it earlier would violate the trust the zero-config posture depends on.

Not planned: bandwidth bonding across peers (real engineering, low payoff next to
just using the cable).
