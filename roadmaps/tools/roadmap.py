#!/usr/bin/env python3
"""roadmap.py — validate and render the knit work-item registry.

Stdlib only (Python 3.11+ for tomllib). Source of truth: roadmaps/registry.toml.

Usage:
  roadmap.py check                 validate the plan; non-zero exit on any error
  roadmap.py render                (re)write roadmaps/STATUS.md from the registry
  roadmap.py order [--milestone M] print a dependency-ordered build sequence
"""
from __future__ import annotations
import sys, tomllib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
REGISTRY = ROOT / "roadmaps" / "registry.toml"
STATUS = ROOT / "roadmaps" / "STATUS.md"

AREAS = {"CORE","PROTO","AUTH","CLI","SYS","DISC","XPORT","AGENT","EXEC",
         "SCHED","CLIENT","AI","OPS","NET","TEST","DOC"}
STATUSES = {"planned","ready","in-progress","review","done","blocked","deferred","dropped"}
DONE = {"done"}
PRIORITIES = {"p0","p1","p2"}
SIZES = {"S","M","L","XL"}
ID_RE_MSG = "KN-<AREA>-<NNN> with a 3+ digit number"


def load():
    with open(REGISTRY, "rb") as f:
        return tomllib.load(f)


def parse_id(iid: str):
    parts = iid.split("-")
    if len(parts) != 3 or parts[0] != "KN":
        return None
    _, area, num = parts
    if area not in AREAS or not (num.isdigit() and len(num) >= 3):
        return None
    return area, num


def validate(data):
    errors, warnings = [], []
    ms = {m["id"]: m for m in data.get("milestone", [])}
    items = data.get("item", [])
    by_id = {}

    if data.get("schema_version") != 1:
        errors.append("schema_version must be 1")

    # milestones
    seen_ms = set()
    for m in data.get("milestone", []):
        if m["id"] in seen_ms:
            errors.append(f"duplicate milestone id {m['id']}")
        seen_ms.add(m["id"])

    # items: identity + enums
    milestone_index = {mid: i for i, mid in enumerate(
        [m["id"] for m in data.get("milestone", []) if m["id"] != "backlog"])}
    for it in items:
        iid = it.get("id", "<missing>")
        if iid in by_id:
            errors.append(f"duplicate item id {iid}")
            continue
        by_id[iid] = it
        p = parse_id(iid)
        if p is None:
            errors.append(f"{iid}: id must be {ID_RE_MSG}")
        elif p[0] != it.get("area"):
            errors.append(f"{iid}: area field '{it.get('area')}' != id area segment '{p[0]}'")
        if it.get("area") not in AREAS:
            errors.append(f"{iid}: unknown area {it.get('area')}")
        if it.get("milestone") not in ms:
            errors.append(f"{iid}: unknown milestone {it.get('milestone')}")
        if it.get("status") not in STATUSES:
            errors.append(f"{iid}: invalid status {it.get('status')}")
        if it.get("priority") not in PRIORITIES:
            errors.append(f"{iid}: invalid priority {it.get('priority')}")
        if it.get("size") not in SIZES:
            errors.append(f"{iid}: invalid size {it.get('size')}")
        if not it.get("accept"):
            errors.append(f"{iid}: at least one acceptance criterion required")
        if not it.get("spec"):
            warnings.append(f"{iid}: no spec doc linked")
        else:
            if not (ROOT / it["spec"]).exists():
                warnings.append(f"{iid}: spec path does not exist: {it['spec']}")

    # dependencies: resolve, self-dep, milestone ordering
    for it in items:
        iid = it["id"]
        for dep in it.get("depends", []):
            if dep == iid:
                errors.append(f"{iid}: depends on itself")
            elif dep not in by_id:
                errors.append(f"{iid}: depends on unknown item {dep}")
            else:
                d = by_id[dep]
                if it.get("status") in DONE and d.get("status") not in DONE:
                    warnings.append(f"{iid} is done but depends on non-done {dep}")
                mi, di = milestone_index.get(it.get("milestone")), milestone_index.get(d.get("milestone"))
                if mi is not None and di is not None and di > mi:
                    warnings.append(f"{iid} ({it['milestone']}) depends on later-milestone {dep} ({d['milestone']})")

    # cycle detection via topological sort
    if not any("depends on unknown" in e for e in errors):
        order, cyc = toposort(by_id)
        if cyc:
            errors.append(f"dependency cycle among: {', '.join(sorted(cyc))}")

    return errors, warnings, by_id, ms


def toposort(by_id):
    indeg = {i: 0 for i in by_id}
    adj = {i: [] for i in by_id}
    for it in by_id.values():
        for dep in it.get("depends", []):
            if dep in by_id:
                adj[dep].append(it["id"])
                indeg[it["id"]] += 1
    # deterministic: pop smallest id among zero-indegree
    ready = sorted([i for i, d in indeg.items() if d == 0])
    order = []
    while ready:
        n = ready.pop(0)
        order.append(n)
        for m in sorted(adj[n]):
            indeg[m] -= 1
            if indeg[m] == 0:
                ready.append(m)
                ready.sort()
    cyc = set(by_id) - set(order)
    return order, cyc


def bar(done, total, width=24):
    if total == 0:
        return "─" * width + "  0/0"
    filled = round(width * done / total)
    return "█" * filled + "░" * (width - filled) + f"  {done}/{total}"


def render(data, by_id, ms):
    order, _ = toposort(by_id)
    order_index = {iid: n for n, iid in enumerate(order)}
    items = list(by_id.values())
    lines = []
    W = lines.append

    W("# knit — plan status")
    W("")
    W("> Generated from `registry.toml` by `roadmaps/tools/roadmap.py render`.")
    W("> Do not hand-edit — change the registry and re-render.")
    W("")

    total_done = sum(1 for it in items if it["status"] in DONE)
    W(f"**{len(items)} items** across **{len([m for m in ms if m!='backlog'])} milestones** "
      f"plus backlog · **{total_done} done**")
    W("")

    # status + priority rollups
    W("## Rollup")
    W("")
    W("| Status | Count |   | Priority | Count |")
    W("| ------ | ----: | - | -------- | ----: |")
    scount = {s: sum(1 for it in items if it["status"] == s) for s in STATUSES}
    pcount = {p: sum(1 for it in items if it["priority"] == p) for p in PRIORITIES}
    srows = [(s, scount[s]) for s in ["ready","in-progress","review","done","planned","blocked","deferred","dropped"] if scount[s]]
    prows = [(p, pcount[p]) for p in ["p0","p1","p2"]]
    for i in range(max(len(srows), len(prows))):
        s = f"{srows[i][0]} | {srows[i][1]}" if i < len(srows) else " | "
        p = f"{prows[i][0]} | {prows[i][1]}" if i < len(prows) else " | "
        W(f"| {s} |   | {p} |")
    W("")

    # per-milestone
    for m in data.get("milestone", []):
        mid = m["id"]
        mine = [it for it in items if it["milestone"] == mid]
        if not mine:
            continue
        d = sum(1 for it in mine if it["status"] in DONE)
        W(f"## {mid} · {m['version']} — {m['name']}")
        W("")
        W(f"*{m['goal']}*")
        W("")
        W("```")
        W(bar(d, len(mine)))
        W("```")
        W("")
        W("| # | ID | Item | Status | Pri | Sz | Depends |")
        W("| -: | -- | ---- | ------ | --- | -- | ------- |")
        for it in sorted(mine, key=lambda x: order_index.get(x["id"], 1e9)):
            deps = ", ".join(it.get("depends", [])) or "—"
            W(f"| {order_index.get(it['id'],'?')} | `{it['id']}` | {it['title']} "
              f"| {it['status']} | {it['priority']} | {it['size']} | {deps} |")
        W("")

    # blocked / attention
    blocked = [it for it in items if it["status"] in {"blocked","deferred"}]
    if blocked:
        W("## Blocked / deferred")
        W("")
        for it in blocked:
            note = it.get("notes", "")
            W(f"- `{it['id']}` **{it['status']}** — {it['title']}" + (f" · {note}" if note else ""))
        W("")

    # suggested build order (m1 happy path)
    W("## Suggested build order (dependency topological sort)")
    W("")
    W("The number in each milestone table is this global order. m1 critical path first:")
    W("")
    m1 = [iid for iid in order if by_id[iid]["milestone"] == "m1"]
    W("```")
    for iid in m1:
        it = by_id[iid]
        W(f"{order_index[iid]:>2}. {iid:<14} {it['priority']} {it['size']}  {it['title']}")
    W("```")
    W("")
    return "\n".join(lines) + "\n"


def cmd_check(data):
    errors, warnings, *_ = validate(data)
    for w in warnings:
        print(f"warning: {w}")
    for e in errors:
        print(f"ERROR: {e}", file=sys.stderr)
    n_items = len(data.get("item", []))
    if errors:
        print(f"\nFAILED: {len(errors)} error(s), {len(warnings)} warning(s) over {n_items} items")
        return 1
    print(f"OK: {n_items} items valid, {len(warnings)} warning(s)")
    return 0


def cmd_render(data):
    errors, warnings, by_id, ms = validate(data)
    if errors:
        print("refusing to render: fix validation errors first (run: roadmap.py check)", file=sys.stderr)
        return 1
    STATUS.write_text(render(data, by_id, ms))
    print(f"wrote {STATUS.relative_to(ROOT)} ({len(by_id)} items)")
    return 0


def cmd_order(data, milestone=None):
    _, _, by_id, _ = validate(data)
    order, cyc = toposort(by_id)
    if cyc:
        print(f"cycle: {cyc}", file=sys.stderr); return 1
    for n, iid in enumerate(order):
        it = by_id[iid]
        if milestone and it["milestone"] != milestone:
            continue
        print(f"{n:>2}. {iid:<14} [{it['milestone']}] {it['priority']} {it['size']}  {it['title']}")
    return 0


def main(argv):
    if len(argv) < 2 or argv[1] in {"-h","--help","help"}:
        print(__doc__); return 0
    data = load()
    cmd = argv[1]
    if cmd == "check":
        return cmd_check(data)
    if cmd == "render":
        return cmd_render(data)
    if cmd == "order":
        ms = None
        if "--milestone" in argv:
            ms = argv[argv.index("--milestone") + 1]
        return cmd_order(data, ms)
    print(f"unknown command: {cmd}\n{__doc__}", file=sys.stderr)
    return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
