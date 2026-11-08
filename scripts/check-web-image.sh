#!/usr/bin/env bash
set -euo pipefail
image=${1:?image is required}
docker run --rm --network none --entrypoint nginx "$image" -t
fixture=$(docker create --network none "$image")
trap 'docker rm -f "$fixture" >/dev/null' EXIT
docker start "$fixture" >/dev/null
for _attempt in {1..20}; do
  if docker exec "$fixture" wget -q -T 2 -O /dev/null http://127.0.0.1/discover; then
    break
  fi
  sleep 0.2
done
docker exec "$fixture" wget -q -T 2 -O - http://127.0.0.1/discover | /usr/bin/grep -q 'Subcult OS'
docker exec "$fixture" wget -q -T 2 -O - http://127.0.0.1/login | /usr/bin/grep -q 'name="robots" content="noindex,nofollow"'
# New private routes must survive the production Nginx allowlist as well as Vite.
for private_path in /door-link/fixture /events/fixture/door-links; do
  docker exec "$fixture" wget -q -T 2 -O - "http://127.0.0.1$private_path" | /usr/bin/grep -q 'name="robots" content="noindex,nofollow"'
done
headers=$(docker exec "$fixture" wget -S -T 2 -O /dev/null http://127.0.0.1/ 2>&1)
/usr/bin/grep -Eq '^[[:space:]]*Location: /discover[[:space:]]*$' <<< "$headers"
headers=$(docker exec "$fixture" wget -S -T 2 -O /dev/null http://127.0.0.1/favicon.ico 2>&1)
/usr/bin/grep -Eq '^[[:space:]]*Content-Type:[[:space:]]*image/png[[:space:]]*$' <<< "$headers"
actual=$(docker exec "$fixture" wget -q -T 2 -O - http://127.0.0.1/favicon.ico | sha256sum)
expected=$(sha256sum web/public/favicon-32.png)
test "${actual%% *}" = "${expected%% *}"
