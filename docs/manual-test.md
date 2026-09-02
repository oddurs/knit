# Manual test matrix (KN-TEST-003)

The automated suite (`go test -race ./...`) proves the protocol and executor over
loopback. The one thing it cannot cover on a single machine is real peer-to-peer
behavior over a physical link. Run this checklist on **two machines** before every
tagged release, once over a **Thunderbolt/USB4 bridge** and once over **Wi-Fi**.

## Setup

On both machines (A and B):

```sh
go build -o knit ./cmd/knit   # or: brew install oddurs/tap/knit
./knit up -d
```

Pair them, one time:

```sh
# on A
./knit key                      # copy the 64 hex chars
# on B
./knit join <key>
```

## Checklist — run once per link (Thunderbolt, then Wi-Fi)

| # | Command (run on A) | Expect |
| - | ------------------ | ------ |
| 1 | `./knit ls` | both machines listed with live load; B shows its LAN/bridge address |
| 2 | `./knit ls --json` | valid JSON array including a `"self":true` entry |
| 3 | `./knit run --on B -- uname -a` | B's uname; one dim `knit → B` line on stderr |
| 4 | `echo hi \| ./knit run --on B -- cat` | prints `hi` (stdin streamed to B) |
| 5 | `./knit run --on B -- sh -c 'exit 7'; echo $?` | prints `7` (exit code relayed) |
| 6 | `head -c 1G /dev/zero \| ./knit run --on B -- wc -c` | prints `1073741824`; note throughput |
| 7 | `./knit run -- <heavy cmd>` with B idle and A busy | schedules onto B automatically |
| 8 | `./knit run --on B -- sleep 60` then Ctrl-C | returns promptly; on B, `pgrep sleep` shows nothing (no orphan) |
| 9 | `./knit each -- hostname` | one prefixed line per machine; exit 0 |
| 10 | `./knit run --on B -- cat bigfile > out` on A | binary-identical (`shasum` matches) |

## Record

For the Thunderbolt run, record the throughput observed in step 6 — it backs the
"line rate" claim in [10-performance.md](10-performance.md). Note the knit
version (`./knit --version`) and both machines' OS/arch.

## Pass criteria

All ten rows pass on both links. Step 8 (no orphaned process after Ctrl-C) and
step 5 (exit code fidelity) are the ones most likely to regress; check them
deliberately.

## Last run

**2026-09-02 · knit v0.2.0 (pre-release build) · cross-host over a virtual bridge**

A = macOS 26 / arm64 (15 cpus), B = Ubuntu / arm64 VM (14 cpus, OrbStack),
reached over the host's bridge interface. B was discovered by mDNS with no
flags; `--peer 192.168.139.118` (default port 5648) was also exercised.

| # | Result |
| - | ------ |
| 1–2 | both machines listed; JSON has `"self":true`; B marked `lan` |
| 3–5 | `uname` from B, stdin `hi` echoed, `exit 7` relayed as 7 |
| 6 | 1 GiB stdin in 0.81 s (~1.3 GB/s, bridge-bound, not a cable number) |
| 7 | `knit run -- hostname` placed on B (lower load per core) |
| 8 | Ctrl-C returned 130 promptly; `pgrep sleep` on B empty |
| 9 | one prefixed line per machine including A; exit 0 |
| 10 | 300 MB `cat` back to A byte-identical (sha256 match) |
| + | `--sync` mirrored a file B wrote back into A's working directory |

Outstanding: the same checklist over a physical Thunderbolt bridge and Wi-Fi,
which is what produces the cable throughput figure for
[10-performance.md](10-performance.md).
