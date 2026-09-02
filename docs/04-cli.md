# CLI surface

Seven commands. If it needs an eighth, the design is drifting.

```
knit up [-d]            start sharing this machine (-d: background)
knit down               stop the background agent
knit gauge [--json]     show machines and their capacity  (alias: ls)
knit run [--on NAME] [--mem GB] [--arch ARCH] [--dir] [--sync] -- CMD [ARGS...]
                        run a command on the machine with most headroom
knit each -- CMD        run a command on every machine at once
knit key                print this machine's cluster key
knit join KEY           join the fabric that key belongs to
```

`--dir` sends the working directory to the target and runs the command there;
`--sync` also mirrors changed files back (implies `--dir`). `--mem GB` and
`--arch` only consider machines with that much memory allocatable right now, or
that architecture, and fail loudly when none qualifies. `--peer host[:port]`
(global, pre-mDNS) adds a machine mDNS cannot see; the port defaults to 5648,
which an agent binds whenever it is free.

## Nomenclature

knit keeps three words from knitting, and each is exact:

| Word | Means |
| ---- | ----- |
| **fabric** | all your machines, working as one |
| **loop** | a single machine in the fabric |
| **gauge** | a machine's capacity — cores, memory, load |

These aren't just flavor: `gauge` is the capacity the scheduler actually reads,
and a loop is exactly one machine. Everything else uses plain words.

The `LINK` column is a best-effort guess from the peer's address — a link-local
address is almost always a Thunderbolt/USB4 bridge (`thunderbolt`), a private
address is a `lan`. True per-interface link speed arrives in v0.3.

## UX rules

- **Silence is the default.** `knit run` writes one dim line to stderr —
  `knit → studio` — only when the command left the machine. Local fallback
  prints nothing. Command output is never wrapped, prefixed, or colored.
- **Exit codes are sacred.** `knit run` exits with the remote command's code.
  Scripts and Makefiles must not be able to tell the difference between local and
  remote execution. knit's own failures use a disjoint high range (see below).
- **stdin/stdout are streams, not buffers.** Pipes work at any size:
  `cat 50GB.log | knit run -- zstd -19 > out.zst`.
- **`each` prefixes, `run` doesn't.** `knit each` interleaves output from many
  machines — this one included — so each line is prefixed `[name] `, and lines
  never split mid-way. It exits non-zero if any machine did, and prints a
  one-line per-machine status summary to stderr at the end.
- **Errors name the fix.** `unauthorized` tells you to run `knit key` /
  `knit join`, not just that auth failed. Every error code in
  [03-protocol.md](03-protocol.md#error-codes) maps to a fix sentence.

## Exit-code contract

The remote command owns codes `0–125` — knit passes them through untouched. For
knit's *own* failures, it uses the shell convention above that range so a script
can distinguish "the command failed" from "knit failed," and never collide with
a real program's exit code.

| Code    | Source                    | Meaning                                        |
| ------- | ------------------------- | ---------------------------------------------- |
| 0–125   | remote (or local) command | the command's own exit status, verbatim        |
| 126     | knit                    | target unreachable / no candidate machine      |
| 127     | knit                    | `unauthorized` — key mismatch with the target  |
| 124     | knit                    | connection dropped before the exit frame       |
| 2       | knit                    | usage error (bad flags, missing `--`)          |

`knit each` exits `0` only if every machine's command exited `0`; otherwise it
exits with the highest knit/command code observed.

## Environment variables

Kept to a minimum; all optional, all overridable by flags where a flag exists.

| Variable            | Effect                                                        |
| ------------------- | ------------------------------------------------------------ |
| `KNIT_HOME`       | override `~/.knit` (key, pidfile, log, peer cache)         |
| `KNIT_PEERS`      | comma-separated `host[:port]` list; adds explicit peers (v0.2) |
| `KNIT_NO_CACHE`   | if set, always browse mDNS fresh (disable the peer cache)    |
| `KNIT_TIMEOUT_MS` | override the per-peer `info` probe budget (default 250)      |
| `NO_COLOR`          | honored — suppresses the dim styling of the `knit →` line  |

`knit each` sets these on every process it launches (this machine is rank 0,
peers follow in order of spare capacity), so multi-node launchers need no
hand-written host list:

| Variable       | Value                                                      |
| -------------- | ---------------------------------------------------------- |
| `KNIT_RANK`    | this machine's position in the launch                      |
| `KNIT_NNODES`  | number of machines in the launch                           |
| `KNIT_HOSTS`   | comma-separated addresses in rank order                    |
| `KNIT_MASTER`  | address of rank 0 (torchrun's `--master_addr`)             |
| `MLX_RANK`     | same as `KNIT_RANK`, as MLX's ring backend reads it        |
| `MLX_HOSTFILE` | path of an MLX-format hostfile written for this launch     |

## Output contracts (for tools built on knit)

- `knit ls --json` emits one JSON array of `info` envelopes plus a synthetic
  entry for the local machine (`"name":"<host>","self":true`). This is the stable
  integration surface the AI launchers in [05-ai-workloads.md](05-ai-workloads.md)
  depend on; fields are only ever added, never renamed or removed within a major
  version.
- The human `knit ls` table is explicitly *not* a stable format — parse
  `--json`, never the table.

## Examples

```sh
# First time, on each machine
knit up -d

# Pair a second machine (one-time)
knit key                      # on machine A → prints 64 hex chars
knit join <key>               # on machine B

# See the fabric
knit ls
#  NAME     ADDR            OS/ARCH        CPUS  MEM     FREE    LOAD  GPU             LINK
#  studio   169.254.87.3    darwin/arm64   24    128.0G  96.4G   0.31  Apple M3 Ultra  thunderbolt ~40G
#  mini     192.168.1.40    darwin/arm64   10    32.0G   11.2G   1.85  Apple M4        wifi
#  here     —               darwin/arm64   8     24.0G   9.8G    4.02  Apple M2        (this machine)

# Offload transparently
knit run -- ffmpeg -i in.mov -c:v libx264 out.mp4

# Pin a machine, or describe what the job needs
knit run --on studio -- python quantize.py
knit run --mem 48 -- python -m mlx_lm.generate --model 70b-q4 ...

# Fan out
knit each -- uname -a

# Multi-node launch: every process gets its rank and the host list
knit each -- python train.py                                   # MLX ring backend
knit each -- sh -c 'torchrun --nnodes $KNIT_NNODES --node_rank $KNIT_RANK \
                    --master_addr $KNIT_MASTER train.py'       # torchrun
```

## Flags kept out on purpose

No `--config`, no `--port`, no daemon-mode client, no output-formatting flags
except `ls --json`. The absence is the point: every flag not added is a decision
a user never has to make.
