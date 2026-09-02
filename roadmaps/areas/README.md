# Area codes

Each area is one concern, and (for the code areas) usually one `internal/` package.
The area code is the middle segment of every item ID: `KN-<AREA>-<NNN>`.

| Code | Area | Maps to | Concern |
| ---- | ---- | ------- | ------- |
| `CORE`   | Core / entrypoint      | `cmd/knit` | `main`, flag dispatch, usage, the exit-code contract |
| `PROTO`  | Wire protocol          | `internal/proto` | frame types, read/write, version token, error codes |
| `AUTH`   | Authentication         | `internal/keys` | keyfile, HMAC sign/verify, TLS pinning (v0.4) |
| `CLI`    | Command surface        | `cmd/knit` | `key`/`join` and CLI/UX contracts |
| `SYS`    | System info            | `internal/sysinfo` | cores, memory, load, GPU/accel (darwin+linux) |
| `DISC`   | Discovery              | `internal/discovery` | mDNS register/browse, peer cache, `--peer` |
| `XPORT`  | Transport              | `internal/transport` | dial, handshake, socket tuning, TLS wrap (v0.4) |
| `AGENT`  | Agent                  | `internal/agent` | listener, connection handler, `up`/`down`, dispatch |
| `EXEC`   | Execution              | `internal/agent` (exec) | spawn, stdio streaming, signals, dir sync |
| `SCHED`  | Scheduler              | `internal/scheduler` | scoring, `--on`/`--mem`/`--arch` filters |
| `CLIENT` | Client commands        | `internal/client` | `ls`, `run`, `each` |
| `AI`     | AI-workload sugar      | `internal/client` (ai) | `hostfile`, `mpirun` |
| `OPS`    | Ops / packaging        | build + CI | static build, release, brew tap, launchd/systemd |
| `NET`    | Network sharing        | `internal/agent` (dial) | `proxy` / SOCKS5 (v0.4) |
| `TEST`   | Test & benchmarks      | `*_test.go`, CI | unit/protocol/e2e/race/fuzz, benches, plan check |
| `DOC`    | Documentation          | `docs/`, repo root | README, SECURITY.md, quickstart |

The numeric block of an ID hints at the milestone (`0xx`=v0.1 … `04x`=v0.4,
`05x+`=backlog), but the item's `milestone` field is authoritative.
