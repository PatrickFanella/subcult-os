#!/usr/bin/env bash
# Exercise `dev-env.sh down` and `gc` stack selection against a mock Docker; no daemon is used.
set -euo pipefail
source_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/live/scripts" "$fixture/state"
cp "$source_root/scripts/dev-env.sh" "$source_root/scripts/dev-stack-gc.py" "$fixture/live/scripts/"
export XDG_STATE_HOME="$fixture/state" DEV_ENV_TRACE="$fixture/trace" FIXTURE="$fixture"
registry="$fixture/state/subcult-os/dev-env/projects"
mkdir -p "$registry"
digest() { printf '%s' "$1" | sha256sum | cut -c1-16; }
live_project="subcult-dev-$(digest "$fixture/live")"
# A stopped stack registered by a since-deleted checkout.
printf '%s\n' "$fixture/gone-registered" > "$registry/subcult-dev-aaaaaaaaaaaaaaaa"

python3 - "$fixture" "$live_project" <<'PY'
import json, pathlib, sys
fixture, live = pathlib.Path(sys.argv[1]), sys.argv[2]
def c(project, wd, files, running=False):
    return {"Id": f"c-{project}", "State": {"Running": running}, "Config": {"Labels": {
        "com.docker.compose.project": project,
        "com.docker.compose.project.working_dir": str(wd),
        "com.docker.compose.project.config_files": ",".join(str(wd / f) for f in files)}}}
containers = [
    c("subcult-dev-6f61b3ecb8", fixture / "gone-old", ["compose.dev.yml"]),
    c("subcult-qa-operator-e25312e", fixture / "gone-qa", ["compose.dev.yml", "override.yml"]),
    c("subcult-review", fixture / "live", ["compose.dev.yml"]),
    c("subcult-dev-bbbbbbbbbbbbbbbb", fixture / "gone-running", ["compose.dev.yml"], running=True),
    c("subcult-os", fixture / "gone-prod", ["docker-compose.yml"]),
    c("subcult-tv", fixture / "gone-other", ["docker-compose.yml"]),
    c("unrelated", fixture / "gone-unrelated", ["compose.dev.yml"]),
]
volumes = [{"Name": f"{p}_dev-data", "Labels": {"com.docker.compose.project": p}} for p in [
    "subcult-dev-6f61b3ecb8", "subcult-dev-aaaaaaaaaaaaaaaa", "subcult-dev-cccccccccccccccc",
    live, "subcult-os", "subcult-wf-14-review"]]
(fixture / "containers.json").write_text(json.dumps(containers))
(fixture / "volumes.json").write_text(json.dumps(volumes))
PY

cat > "$fixture/bin/docker" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DEV_ENV_TRACE"
query() {
  python3 - "$FIXTURE/$1.json" "$2" "$3" <<'PY'
import json, sys
items, mode, arg = json.load(open(sys.argv[1])), sys.argv[2], sys.argv[3]
def project(i):
    return (i.get("Config", {}).get("Labels") or i.get("Labels"))["com.docker.compose.project"]
def key(i):
    return i.get("Id") or i["Name"]
if mode == "ids":
    rest = arg.removeprefix("label=")
    want = rest.split("=", 1)[1] if "=" in rest else None
    print("\n".join(key(i) for i in items if want is None or project(i) == want))
else:
    print(json.dumps([i for i in items if key(i) in arg.split()]))
PY
}
case "$1 $2" in
  "ps -aq") query containers ids "$4" ;;
  "volume ls") query volumes ids "$5" ;;
  "container inspect") shift 2; query containers inspect "$*" ;;
  "volume inspect") shift 2; query volumes inspect "$*" ;;
esac
SH
chmod +x "$fixture/bin/docker"
export PATH="$fixture/bin:$PATH"

report=$(bash "$fixture/live/scripts/dev-env.sh" gc)
printf '%s\n' "$report"
! grep -Eq ' (rm|volume rm|network rm) ' "$DEV_ENV_TRACE" || { echo 'report mode removed resources' >&2; exit 1; }
expect() { grep -Eq "^$1 +$2 " <<<"$report" || { echo "expected $1 for $2" >&2; exit 1; }; }
expect remove subcult-dev-6f61b3ecb8
expect remove subcult-qa-operator-e25312e
expect remove subcult-dev-aaaaaaaaaaaaaaaa
expect orphan subcult-dev-cccccccccccccccc
expect keep subcult-review
expect keep subcult-dev-bbbbbbbbbbbbbbbb
expect keep subcult-os
expect keep subcult-tv
expect keep "$live_project"
expect keep subcult-wf-14-review
! grep -q unrelated <<<"$report" || { echo 'reported a non-subcult project' >&2; exit 1; }

: > "$DEV_ENV_TRACE"
bash "$fixture/live/scripts/dev-env.sh" gc --apply --orphans --project subcult-review >/dev/null
removed=$(grep -E '^(rm|volume rm|network rm) ' "$DEV_ENV_TRACE")
for name in subcult-dev-6f61b3ecb8 subcult-qa-operator-e25312e subcult-dev-aaaaaaaaaaaaaaaa \
    subcult-dev-cccccccccccccccc subcult-review; do
  grep -q -- "$name" <<<"$removed" || { echo "did not remove $name" >&2; exit 1; }
done
for name in subcult-dev-bbbbbbbbbbbbbbbb subcult-os subcult-tv "$live_project" subcult-wf-14-review unrelated; do
  ! grep -q -- "$name" <<<"$removed" || { echo "removed protected stack $name" >&2; exit 1; }
done
[[ ! -e "$registry/subcult-dev-aaaaaaaaaaaaaaaa" ]] || { echo 'registry entry kept after removal' >&2; exit 1; }

: > "$DEV_ENV_TRACE"
printf '%s\n' "$fixture/live" > "$registry/$live_project"
bash "$fixture/live/scripts/dev-env.sh" down
grep -q -- "-p $live_project -f $fixture/live/compose.dev.yml --profile test down -v --remove-orphans" "$DEV_ENV_TRACE"
[[ ! -e "$registry/$live_project" ]] || { echo 'down kept the registry entry' >&2; exit 1; }
echo 'Dev stack down and gc selection passed.'
