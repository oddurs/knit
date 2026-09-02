# connex roadmaps

The plan is not prose you have to keep in sync by hand. It is a **coded system**:
every unit of work has a stable ID, lives in one machine-checkable registry, and
is rendered into human roadmaps. The registry is the source of truth; the milestone
docs are the readable view; a small validator keeps them honest.

```
roadmaps/
  README.md                 ← you are here: the coding system + how to use it
  registry.toml             ← source of truth: every work item, coded
  STATUS.md                 ← generated snapshot (do not hand-edit)
  milestones/               ← human narrative per release, referencing item IDs
    m1-v0.1-fabric.md
    m2-v0.2-real-use.md
    m3-v0.3-ai-native.md
    m4-v0.4-encrypted-persistent.md
  areas/
    README.md               ← what each area code means
    backlog.md              ← deferred / unscheduled items
  tools/
    roadmap.py              ← stdlib-only validator + renderer (Python 3.11+)
    README.md               ← how to run it
```

## The item ID scheme

Every work item has an ID of the form:

```
CX-<AREA>-<NNN>
   │      │
   │      └─ zero-padded sequence within the area (001, 010, 030 …)
   └──────── area code (see areas/README.md)
```

The numeric block is loosely milestone-aligned so an ID hints at when it lands:
`0xx` = v0.1, `02x` = v0.2, `03x` = v0.3, `04x` = v0.4, `05x+` = backlog. This is a
convention for readability, not a rule the validator enforces — `milestone` is the
authoritative field.

Examples: `CX-EXEC-001` (core executor, v0.1), `CX-SYS-030` (free-mem/GPU reporting,
v0.3), `CX-NET-040` (SOCKS proxy, v0.4).

## Area codes

Sixteen areas, each mapping to one concern (and usually one `internal/` package).
Full descriptions in [areas/README.md](areas/README.md).

`CORE` `PROTO` `AUTH` `CLI` `SYS` `DISC` `XPORT` `AGENT` `EXEC` `SCHED` `CLIENT`
`AI` `OPS` `NET` `TEST` `DOC`

## Status lifecycle

```
planned ──▶ ready ──▶ in-progress ──▶ review ──▶ done
   │           │            │
   └───────────┴────────────┴────────▶ blocked      (waiting on a dependency/decision)
                                       deferred     (intentionally not now)
                                       dropped      (decided against; kept for the record)
```

- **planned** — captured, not yet refined.
- **ready** — refined, dependencies satisfied, safe to start.
- **in-progress / review / done** — the usual flow.
- **blocked / deferred / dropped** — off the happy path, always with a `notes` reason.

## Priority and size

- **priority**: `p0` (blocks its milestone) · `p1` (should make it) · `p2` (nice to have).
- **size**: `S` (< ½ day) · `M` (~1–2 days) · `L` (~3–5 days) · `XL` (split it).

## Dependencies

Each item lists `depends = [ids]`. The validator resolves every ID, forbids
self-dependency, and topologically sorts the graph to **prove it is acyclic** and to
emit a suggested build order. It also warns when a `done` item depends on something
not `done`, or when an item depends on work scheduled in a *later* milestone (a
sequencing smell).

## How to use it

```sh
# Validate the whole plan (run in CI; non-zero exit on any error)
python3 roadmaps/tools/roadmap.py check

# Regenerate STATUS.md from the registry
python3 roadmaps/tools/roadmap.py render

# Print the dependency-ordered build sequence for a milestone
python3 roadmaps/tools/roadmap.py order --milestone m1
```

The registry is edited by hand (it is plain TOML); the milestone docs and
`STATUS.md` are the views. When you change an item, change it in `registry.toml` and
re-render. CI fails if `STATUS.md` is stale or the graph is invalid, so the plan can
never quietly drift from reality — the same standard the code is held to.

## Milestones at a glance

| Milestone | Version | Theme | Detail |
| --------- | ------- | ----- | ------ |
| m1 | v0.1 | The fabric — `run`/`each`/`ls` end to end | [m1](milestones/m1-v0.1-fabric.md) |
| m2 | v0.2 | Trustworthy under real use | [m2](milestones/m2-v0.2-real-use.md) |
| m3 | v0.3 | AI-native scheduling | [m3](milestones/m3-v0.3-ai-native.md) |
| m4 | v0.4 | Encrypted, persistent, shareable | [m4](milestones/m4-v0.4-encrypted-persistent.md) |
| backlog | — | Measured future wins, deferred by choice | [backlog](areas/backlog.md) |
