#!/usr/bin/env bash
# Exercise checkout identity, failure cleanup and network release without running Docker or mise.
set -euo pipefail
source_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/checkout-a/scripts" "$fixture/checkout-b/scripts"
cp "$source_root/scripts/dev-env.sh" "$fixture/checkout-a/scripts/"
cp "$source_root/scripts/dev-env.sh" "$fixture/checkout-b/scripts/"
export DEV_ENV_TRACE="$fixture/trace"
cat > "$fixture/bin/docker" <<'SH'
#!/usr/bin/env bash
printf 'docker\0' >> "$DEV_ENV_TRACE"
printf '%s\0' "$@" >> "$DEV_ENV_TRACE"
printf '\n' >> "$DEV_ENV_TRACE"
if [[ " $* " == *' port test-db 5432 '* ]]; then
  printf '127.0.0.1:35432\n'
fi
if [[ " $* " == *' ps -aq '* && "${DEV_ENV_PREVIEW_PRESENT:-}" == 1 ]]; then
  printf 'preview-container\n'
fi
if [[ " $* " == *' logs --no-color test-db '* && "${DEV_ENV_FAIL_LOGS:-}" == 1 ]]; then
  exit 17
fi
SH
cat > "$fixture/bin/mise" <<'SH'
#!/usr/bin/env bash
printf 'mise\0' >> "$DEV_ENV_TRACE"
printf '%s\0' "$@" >> "$DEV_ENV_TRACE"
printf '\n' >> "$DEV_ENV_TRACE"
if [[ " $* " == *' make test-db '* ]]; then
  exit 23
fi
SH
cat > "$fixture/bin/flock" <<'SH'
#!/usr/bin/env bash
# Avoid taking the real host lock in this mock-only regression.
exit 0
SH
chmod +x "$fixture/bin/"*
export PATH="$fixture/bin:$PATH"
export COMPOSE_FILE=/invalid/deployment.yml COMPOSE_PROJECT_NAME=production
export DEV_WEB_URL=https://invalid.example.test
bash "$fixture/checkout-a/scripts/dev-env.sh" status
bash "$fixture/checkout-b/scripts/dev-env.sh" status
set +e
DEV_ENV_PREVIEW_PRESENT=1 bash "$fixture/checkout-a/scripts/dev-env.sh" test-db
result=$?
set -e
[[ "$result" == 23 ]] || { echo "DB failure became exit $result instead of 23" >&2; exit 1; }
set +e
DEV_ENV_FAIL_LOGS=1 bash "$fixture/checkout-a/scripts/dev-env.sh" test-db 2> "$fixture/log-failure.err"
result=$?
set -e
[[ "$result" == 23 ]] || { echo "Log failure changed the test exit to $result" >&2; exit 1; }
[[ "$(cat "$fixture/log-failure.err")" == 'Could not retain test database logs.' ]]
bash "$fixture/checkout-a/scripts/dev-env.sh" stop
python3 - "$DEV_ENV_TRACE" "$fixture" <<'PY'
import pathlib, sys
trace, fixture = map(pathlib.Path, sys.argv[1:])
commands = [row.decode().split('\0')[:-1] for row in trace.read_bytes().split(b'\n') if row]
docker = [row[1:] for row in commands if row[0] == 'docker']
roots = [str(fixture / name) for name in ('checkout-a', 'checkout-b')]
projects = [row[row.index('-p') + 1] for row in docker[:2]]
assert len(set(projects)) == 2 and all(p.startswith('subcult-dev-') for p in projects)
for row in docker:
    root = row[row.index('--project-directory') + 1]
    assert root in roots
    assert row[row.index('--env-file') + 1] == '/dev/null'
    assert row[row.index('-f') + 1] == root + '/compose.dev.yml'
    assert 'production' not in row and '/invalid/deployment.yml' not in row
removals = [i for i, row in enumerate(docker) if row[-3:] == ['rm', '-sf', 'test-db']]
assert len(removals) == 2, 'a failed test did not clean up its test DB'
downs = [i for i, row in enumerate(docker) if row[-1] == 'down']
# With preview containers present the network stays; an idle project releases it; stop releases it.
assert downs == [removals[1] + 2, len(docker) - 1], 'network release ran at the wrong time'
assert docker[removals[0] + 1][-2:] == ['ps', '-aq'] and docker[removals[1] + 1][-2:] == ['ps', '-aq']
assert not any('postgres' in row or '-v' in row or '--volumes' in row for row in docker), 'cleanup affected preview data'
print('Dev environment checkout isolation, failed-test cleanup and network release passed.')
PY
