#!/usr/bin/env bash
set -euo pipefail

project_name="${COMPOSE_PROJECT_NAME:-subcult-os}"
api_url="${API_URL:-}"

if [[ -z "${api_url}" ]]; then
  api_port="$(docker compose -p "${project_name}" port api 8080 | awk -F: '{print $NF}')"
  if [[ -z "${api_port}" ]]; then
    echo "Could not determine API port. Is the stack running? Try: make up-build" >&2
    exit 1
  fi
  api_url="http://localhost:${api_port}"
fi

# shellcheck source=scripts/qa-identity.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/qa-identity.sh"
qa_require_disposable_target

tmpdir="$(mktemp -d /tmp/subcult-fake-event-qa.XXXXXX)"
trap 'rm -rf "${tmpdir}"' EXIT

cookie="${tmpdir}/organizer.cookies"
suffix="$(date +%s%N)"
email="organizer+${suffix}@example.test"

json_get() {
  python -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$1"
}

json_bool() {
  python -c 'import json,sys; print("true" if json.load(sys.stdin)[sys.argv[1]] else "false")' "$1"
}

echo "Fake-event organizer QA against ${api_url}"

curl -fsS -c "${cookie}" -H 'Content-Type: application/json' \
  -d "{\"email\":\"${email}\",\"password\":\"secret1234\",\"displayName\":\"Fake Event Organizer\"}" \
  "${api_url}/api/auth/signup" >/dev/null
qa_verify_signup "$cookie" "$email"
echo "✓ organizer account created and verified"

workspace_json="$(curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{"name":"Fake Event Rehearsal Workspace"}' \
  "${api_url}/api/workspaces")"
workspace_id="$(json_get id <<<"${workspace_json}")"
echo "✓ workspace created"

starts_at="$(python - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) + timedelta(days=10)).replace(microsecond=0).isoformat().replace('+00:00', 'Z'))
PY
)"

event_json="$(curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d "{\"title\":\"Fake Event Rehearsal\",\"startsAt\":\"${starts_at}\",\"publicDescription\":\"A rehearsal event for organizer setup flow.\",\"locationDisplay\":\"Mobile rehearsal room\",\"imageUrl\":\"https://images.unsplash.com/photo-1492684223066-81342ee5ff30\",\"ticketAllocation\":25,\"pricingMode\":\"free\",\"ticketPriceCents\":0,\"ticketCurrency\":\"usd\"}" \
  "${api_url}/api/workspaces/${workspace_id}/events")"
event_id="$(json_get id <<<"${event_json}")"
echo "✓ draft event created"

published_json="$(curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{}' \
  "${api_url}/api/events/${event_id}/publish")"
slug="$(json_get publicSlug <<<"${published_json}")"
echo "✓ event published (${slug})"

test_ticket_json="$(curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{}' \
  "${api_url}/api/events/${event_id}/test-ticket")"
ticket_code="$(json_get code <<<"${test_ticket_json}")"
echo "✓ organizer test ticket created"

public_role_json="$(curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{"name":"Public performer","description":"Applicant-facing role.","capacity":2,"public":true}' \
  "${api_url}/api/events/${event_id}/roles")"
public_role_id="$(json_get id <<<"${public_role_json}")"
private_role_json="$(curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{"name":"Internal runner","description":"Organizer-only role.","capacity":1,"public":false}' \
  "${api_url}/api/events/${event_id}/roles")"
private_role_id="$(json_get id <<<"${private_role_json}")"
echo "✓ public and private roles created"

updated_private_role="$(curl -fsS -X PATCH -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{"name":"Internal setup runner","description":"Organizer-only setup role.","capacity":1,"public":false,"active":true}' \
  "${api_url}/api/events/${event_id}/roles/${private_role_id}")"
python -c 'import json,sys; r=json.load(sys.stdin); assert r["name"] == "Internal setup runner" and not r["public"] and r["active"], r; print("✓ private role edited")' <<<"${updated_private_role}"

public_roles_json="$(curl -fsS "${api_url}/api/public/events/${slug}/roles")"
PUBLIC_ROLE_ID="${public_role_id}" python -c 'import json,os,sys; roles=json.load(sys.stdin); assert len(roles) == 1 and roles[0]["id"] == os.environ["PUBLIC_ROLE_ID"], roles; print("✓ public roles only expose active public role")' <<<"${public_roles_json}"

application_json="$(curl -fsS -H 'Content-Type: application/json' \
  -d "{\"roleId\":\"${public_role_id}\",\"applicantName\":\"Fake Applicant\",\"applicantEmail\":\"applicant-${suffix}@example.test\",\"message\":\"Happy to help.\"}" \
  "${api_url}/api/public/events/${slug}/role-applications")"
application_id="$(json_get id <<<"${application_json}")"
curl -fsS -X PATCH -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{"status":"accepted"}' \
  "${api_url}/api/events/${event_id}/role-applications/${application_id}" >/dev/null
echo "✓ public application submitted and accepted"

staffing_json="$(curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{"title":"Open doors","kind":"task","notes":"Set up QR scanner.","startsAt":null,"endsAt":null}' \
  "${api_url}/api/events/${event_id}/staffing")"
staffing_id="$(json_get id <<<"${staffing_json}")"
updated_staffing_json="$(curl -fsS -X PATCH -b "${cookie}" -H 'Content-Type: application/json' \
  -d '{"title":"Open doors and test scanner","notes":"Set up QR scanner and backup lookup.","clearStartsAt":true,"clearEndsAt":true}' \
  "${api_url}/api/events/${event_id}/staffing/${staffing_id}")"
python -c 'import json,sys; item=json.load(sys.stdin); assert item["title"] == "Open doors and test scanner", item; print("✓ run-of-show item edited")' <<<"${updated_staffing_json}"

search_json="$(curl -fsS -b "${cookie}" "${api_url}/api/events/${event_id}/door/tickets?query=${ticket_code}")"
python -c 'import json,sys; data=json.load(sys.stdin); assert len(data) == 1, data; print("✓ test ticket found by door search")' <<<"${search_json}"

curl -fsS -b "${cookie}" -H 'Content-Type: application/json' \
  -d "{\"code\":\"${ticket_code}\"}" \
  "${api_url}/api/events/${event_id}/door/check-ins" >/dev/null
echo "✓ test ticket checked in"

event_after_json="$(curl -fsS -b "${cookie}" "${api_url}/api/events/${event_id}")"
python -c 'import json,sys; event=json.load(sys.stdin); assert event["reservedCount"] >= 1, event; assert event["checkedInCount"] >= 1, event; assert event["staffingOpenCount"] + event["staffingAssignedCount"] + event["staffingCompletedCount"] + event["staffingCancelledCount"] >= 1, event; print("✓ event dashboard counts reflect rehearsal setup")' <<<"${event_after_json}"

echo "Fake-event organizer QA passed."
