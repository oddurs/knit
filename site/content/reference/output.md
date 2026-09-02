---
title: Output formats
order: 4
---
# Output formats

`knit gauge --json` is the one machine-readable surface. Fields are only ever
added within a major version, never renamed or removed, so scripts can rely on
it.

```sh
knit gauge --json
```

```json
[
  {
    "ok": true,
    "name": "here",
    "os": "darwin",
    "arch": "arm64",
    "cpus": 8,
    "mem_gb": 24,
    "mem_free_gb": 9.8,
    "load1": 4.02,
    "accel": "metal",
    "gpu": "Apple M2",
    "self": true
  },
  {
    "ok": true,
    "name": "studio",
    "os": "darwin",
    "arch": "arm64",
    "cpus": 24,
    "mem_gb": 128,
    "mem_free_gb": 96.4,
    "load1": 0.31,
    "accel": "metal",
    "gpu": "Apple M3 Max",
    "link": "thunderbolt ~40G"
  }
]
```

| Field | Meaning |
| ----- | ------- |
| `name` | short hostname, the value `--on` takes |
| `os`, `arch` | `darwin` or `linux`; `arm64` or `amd64` |
| `cpus` | logical cores |
| `mem_gb` | total memory, GiB, one decimal |
| `mem_free_gb` | allocatable now; what `--mem` compares against |
| `load1` | one-minute load average |
| `accel` | `metal`, `cuda`, or `none` |
| `gpu` | chip or card name; absent when unknown |
| `self` | `true` on this machine's entry only |
| `link` | how this machine is reached; absent on the self entry |

Fields with zero or empty values may be omitted. Machines that did not answer
the probe, or refused the key, are not in the list.

Everything else knit prints is for people: the `knit → name` line and the
`knit each:` summary go to stderr and may change wording.
