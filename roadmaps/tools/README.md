# roadmap.py

A stdlib-only (Python 3.11+) validator and renderer for the connex work-item
registry. No dependencies — it uses `tomllib` from the standard library.

## Commands

```sh
python3 roadmaps/tools/roadmap.py check
```
Validates the whole plan and exits non-zero on any error. Checks:
- `schema_version` is 1; milestone IDs unique.
- every item ID matches `CX-<AREA>-<NNN>` and its area segment equals its `area`.
- `area`, `milestone`, `status`, `priority`, `size` are from the allowed sets.
- every item has at least one acceptance criterion; linked `spec` paths exist.
- every dependency resolves; nothing depends on itself; the graph is **acyclic**.
- warns when a `done` item depends on a non-done one, or on a later-milestone item.

```sh
python3 roadmaps/tools/roadmap.py render
```
Regenerates [`../STATUS.md`](../STATUS.md) — rollups, per-milestone progress bars and
tables, blocked/deferred list, and the dependency-sorted build order. Refuses to run
if validation fails.

```sh
python3 roadmaps/tools/roadmap.py order [--milestone m1]
```
Prints the topologically-sorted build sequence, optionally filtered to one milestone.

## In CI

`check` runs in the pipeline ([`CX-TEST-002`](../registry.toml)) so an invalid or
drifted plan fails the build the same way broken code does. Re-run `render` and commit
`STATUS.md` whenever the registry changes.
