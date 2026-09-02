# Manual test matrix (CX-TEST-003)

The automated suite (`go test -race ./...`) proves the protocol and executor over
loopback. The one thing it cannot cover on a single machine is real peer-to-peer
behavior over a physical link. Run this checklist on **two machines** before every
tagged release, once over a **Thunderbolt/USB4 bridge** and once over **Wi-Fi**.

## Setup

On both machines (A and B):

```sh
go build -o connex ./cmd/connex   # or: brew install oddurs/tap/connex (v0.2+)
./connex up -d
```

Pair them, one time:

```sh
# on A
./connex key                      # copy the 64 hex chars
# on B
./connex join <key>
```

## Checklist — run once per link (Thunderbolt, then Wi-Fi)

| # | Command (run on A) | Expect |
| - | ------------------ | ------ |
| 1 | `./connex ls` | both machines listed with live load; B shows its LAN/bridge address |
| 2 | `./connex ls --json` | valid JSON array including a `"self":true` entry |
| 3 | `./connex run --on B -- uname -a` | B's uname; one dim `connex → B` line on stderr |
| 4 | `echo hi \| ./connex run --on B -- cat` | prints `hi` (stdin streamed to B) |
| 5 | `./connex run --on B -- sh -c 'exit 7'; echo $?` | prints `7` (exit code relayed) |
| 6 | `head -c 1G /dev/zero \| ./connex run --on B -- wc -c` | prints `1073741824`; note throughput |
| 7 | `./connex run -- <heavy cmd>` with B idle and A busy | schedules onto B automatically |
| 8 | `./connex run --on B -- sleep 60` then Ctrl-C | returns promptly; on B, `pgrep sleep` shows nothing (no orphan) |
| 9 | `./connex each -- hostname` | one prefixed line per machine; exit 0 |
| 10 | `./connex run --on B -- cat bigfile > out` on A | binary-identical (`shasum` matches) |

## Record

For the Thunderbolt run, record the throughput observed in step 6 — it backs the
"line rate" claim in [10-performance.md](10-performance.md). Note the connex
version (`./connex --version`) and both machines' OS/arch.

## Pass criteria

All ten rows pass on both links. Step 8 (no orphaned process after Ctrl-C) and
step 5 (exit code fidelity) are the ones most likely to regress; check them
deliberately.
