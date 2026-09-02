---
title: Choose the right machine
order: 4
---
# Choose the right machine

By default `knit run` goes to the machine with the lowest load per core, this
one included. Three flags shape that.

## Pin one: --on

```sh
knit run --on studio -- ...
```

No scoring; the command goes there, or fails with exit code 126 if `studio`
cannot be reached.

## Require memory: --mem

```sh
knit run --mem 48 -- python -m mlx_lm.generate --model 70b-q4 ...
```

Only machines with at least 48 GB allocatable right now are considered; the
least loaded of those wins. "Allocatable" is what a process could actually take
this moment, not total RAM: on macOS, total minus wired, compressed, and app
memory, with file cache counted as free; on Linux, the kernel's `MemAvailable`.
`knit gauge` shows it in the FREE column.

If no machine qualifies, knit refuses and says which came closest:

```
knit: no machine has 48 GB free (most: studio with 31.2 GB)
```

It never falls back to running a job locally that would not fit.

## Require an architecture: --arch

```sh
knit run --arch amd64 -- ./vendor-tool
```

For native binaries that only exist for one architecture. Values are Go's
names: `arm64`, `amd64`.

Flags combine. `--on` is checked against the machines that pass `--mem` and
`--arch`, so `--on studio --mem 48` fails rather than run on a studio that
cannot hold the job.

## How ties are broken

Two machines within a hundredth of a load-per-core are a tie. Ties go to the
faster link, so a machine reached over a Thunderbolt cable beats the same
machine reached over Wi-Fi, and this machine, needing no link at all, beats
either. A machine that is clearly less loaded always wins regardless of link.

## What the score does not know

Load per core is a one-minute average, so a machine that just finished
something heavy looks busy for a little while. The probe is live on every run,
never cached, so the picture is at most a minute stale.
