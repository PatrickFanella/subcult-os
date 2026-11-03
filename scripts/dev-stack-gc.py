#!/usr/bin/env python3
"""Find and remove development Compose stacks whose checkout no longer exists.

Invoked by `bash scripts/dev-env.sh gc`. Without --apply it only reports.

A stack is a candidate only when it is a development stack: its project name
starts with `subcult-dev-`, or every container was created from a
`compose.dev.yml`. Deployment stacks (`subcult-os`, or any project that uses
`docker-compose.yml`) are never candidates. Stacks with a running container
are never removed. Volume-only `subcult-dev-*` stacks are matched against the
registry written by `dev-env.sh start` and against this repository's existing
worktrees; one with no known owner is removed only with --orphans. --project
removes one named non-deployment `subcult-*` stack, for example a disposable
review stack whose checkout still exists.
"""
import argparse
import hashlib
import json
import subprocess
import sys
from pathlib import Path

PROJECT = "com.docker.compose.project"
WORKING_DIR = "com.docker.compose.project.working_dir"
CONFIG_FILES = "com.docker.compose.project.config_files"
DEV_PREFIX = "subcult-dev-"
PROTECTED = {"subcult-os"}


def docker(*args):
    return subprocess.run(["docker", *args], check=True, stdout=subprocess.PIPE, text=True).stdout


def ids(*args):
    return docker(*args).split()


def inspect(kind, names):
    return json.loads(docker(*kind, "inspect", *names)) if names else []


def dev_project_name(root):
    digest = hashlib.sha256(str(root).encode()).hexdigest()
    return DEV_PREFIX + digest[:16]


def live_worktrees(repo):
    """Project names for this repository's existing worktrees (best effort)."""
    try:
        out = subprocess.run(["git", "-C", str(repo), "worktree", "list", "--porcelain"],
                             check=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True).stdout
    except (OSError, subprocess.CalledProcessError):
        out = ""
    roots = {Path(line[len("worktree "):]) for line in out.splitlines() if line.startswith("worktree ")}
    roots.add(Path(repo))
    return {dev_project_name(root.resolve()) for root in roots if (root / "scripts/dev-env.sh").is_file()}


def collect():
    stacks = {}
    for c in inspect(["container"], ids("ps", "-aq", "--filter", f"label={PROJECT}")):
        labels = c["Config"].get("Labels") or {}
        stack = stacks.setdefault(labels[PROJECT], {"containers": [], "volumes": []})
        stack["containers"].append({
            "id": c["Id"],
            "running": c["State"].get("Running", False),
            "working_dir": labels.get(WORKING_DIR, ""),
            "config_files": [f for f in labels.get(CONFIG_FILES, "").split(",") if f],
        })
    for v in inspect(["volume"], ids("volume", "ls", "-q", "--filter", f"label={PROJECT}")):
        labels = v.get("Labels") or {}
        stacks.setdefault(labels[PROJECT], {"containers": [], "volumes": []})["volumes"].append(v["Name"])
    return stacks


def classify(name, stack, registry, live):
    """Return (action, reason); action is 'remove', 'orphan' or 'keep'."""
    containers = stack["containers"]
    files = [Path(f).name for c in containers for f in c["config_files"]]
    if name in PROTECTED or "docker-compose.yml" in files:
        return "keep", "deployment stack"
    is_dev_name = name.startswith(DEV_PREFIX)
    if containers:
        if not is_dev_name and not all("compose.dev.yml" in [Path(f).name for f in c["config_files"]]
                                       for c in containers):
            return "keep", "not a compose.dev.yml stack"
        if any(c["running"] for c in containers):
            return "keep", "has running containers"
        dirs = {c["working_dir"] for c in containers}
        if "" in dirs:
            return "keep", "container has no working directory label"
        present = sorted(d for d in dirs if Path(d).exists())
        if present:
            return "keep", f"checkout exists: {present[0]}"
        return "remove", f"checkout removed: {min(dirs)}"
    if not is_dev_name:
        return "keep", "volume-only stack without a dev name"
    owner = registry / name
    if owner.is_file():
        root = Path(owner.read_text().strip())
        if (root / "scripts/dev-env.sh").is_file():
            return "keep", f"checkout exists: {root}"
        return "remove", f"checkout removed: {root}"
    if name in live:
        return "keep", "matches an existing worktree"
    return "orphan", "volumes only; owning checkout unknown"


def remove(name):
    containers = ids("ps", "-aq", "--filter", f"label={PROJECT}={name}")
    if containers:
        docker("rm", "-f", "-v", *containers)
    volumes = ids("volume", "ls", "-q", "--filter", f"label={PROJECT}={name}")
    if volumes:
        docker("volume", "rm", *volumes)
    networks = ids("network", "ls", "-q", "--filter", f"label={PROJECT}={name}")
    if networks:
        docker("network", "rm", *networks)


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--apply", action="store_true", help="remove stale stacks (default: report only)")
    parser.add_argument("--orphans", action="store_true",
                        help="also remove volume-only subcult-dev-* stacks whose checkout is unknown")
    parser.add_argument("--project", action="append", default=[],
                        help="remove this named subcult-* stack even if its checkout exists")
    parser.add_argument("--repo", default=Path(__file__).resolve().parent.parent, type=Path)
    parser.add_argument("--registry", type=Path, required=True)
    args = parser.parse_args()

    live = live_worktrees(args.repo)
    plan = []
    stacks = collect()
    for name in args.project:
        if not name.startswith("subcult-") or name not in stacks:
            parser.error(f"no subcult-* Compose stack named {name}")
    for name, stack in sorted(stacks.items()):
        if not name.startswith("subcult-"):
            continue  # Other projects on the host are never ours.
        action, reason = classify(name, stack, args.registry, live)
        if action == "orphan" and args.orphans:
            action = "remove"
        if name in args.project and reason not in ("deployment stack", "has running containers"):
            action, reason = "remove", "requested with --project"
        plan.append((name, action, reason, len(stack["containers"]), len(stack["volumes"])))

    for name, action, reason, nc, nv in plan:
        print(f"{action:6} {name} ({nc} containers, {nv} volumes): {reason}")
    stale = [name for name, action, *_ in plan if action == "remove"]
    if not args.apply:
        if stale:
            print(f"\n{len(stale)} stale stack(s). Re-run with --apply to remove them.")
        return 0
    for name in stale:
        remove(name)
        (args.registry / name).unlink(missing_ok=True)
        print(f"removed {name}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
