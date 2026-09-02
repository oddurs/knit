---
title: AI workloads
order: 8
---
# AI workloads

Two Macs on a desk with a Thunderbolt cable between them can serve a model that
neither holds alone. The tooling for that exists; what it always needs is the
same three answers: which machines, what memory and accelerator, how to start a
process on each. knit answers all three.

## See what you have

```sh
knit gauge
```

FREE is memory a model could take right now; GPU names the chip or card.
`knit gauge --json` gives the same as JSON, with `mem_free_gb`, `accel`
(`metal`, `cuda`, or `none`), and `gpu`, for scripts that plan a split.

## Put a job where it fits

```sh
knit run --mem 48 -- python -m mlx_lm.generate --model mlx-community/... --prompt "..."
```

Refuses every machine with less than 48 GB allocatable and picks the least
loaded of the rest. Nothing to look up first.

## Launch across machines

`knit each` starts a process on every machine and gives each one its rank, the
number of machines, the host list in rank order, and the address of rank 0. The
machine you run it from is rank 0; the others follow by spare capacity.

### MLX distributed

MLX's ring backend reads `MLX_RANK` and `MLX_HOSTFILE`. knit sets both, writing
a hostfile for the launch on each machine, so:

```sh
knit each -- python train.py
```

is the whole launch. Inside, `mx.distributed.init()` finds its peers.

### torchrun

```sh
knit each -- sh -c 'torchrun --nnodes $KNIT_NNODES --node_rank $KNIT_RANK \
                     --master_addr $KNIT_MASTER --master_port 29500 train.py'
```

The `sh -c` is there so the variables are expanded on each machine, not by your
local shell.

### llama.cpp RPC

```sh
knit each -- sh -c 'rpc-server -H 0.0.0.0 -p 50052'      # workers everywhere
llama-cli -m 70b-q4.gguf --rpc studio.local:50052,mini.local:50052 -ngl 99
```

Leave the first command running in one terminal; run the client in another.

## The honest physics

Tensor-parallel inference moves activations between machines for every token
and is bandwidth-bound: excellent over a Thunderbolt cable, disappointing over
Wi-Fi. Pipeline and layer splits are far more forgiving. `knit gauge` shows the
link so you can choose with eyes open, and knit dials the cable when there is
one.

## Batch work is the easy win

Quantizing, embedding, evaluating, and converting are "one heavy command, big
I/O" jobs. knit streams them well and needs no launcher at all:

```sh
knit run --on studio -- python -m mlx_lm.convert --hf-path meta-llama/... -q
cat corpus.jsonl | knit run -- python embed.py > vectors.jsonl
```
