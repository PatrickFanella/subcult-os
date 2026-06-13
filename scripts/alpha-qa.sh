#!/usr/bin/env bash
set -euo pipefail

paid_mode=false
while (($#)); do
  case "$1" in
    --paid)
      paid_mode=true
      ;;
    -h|--help)
      echo "Usage: $0 [--paid]" >&2
      exit 0
      ;;
    *)
      echo "Usage: $0 [--paid]" >&2
      exit 1
      ;;
  esac
  shift
done

project_name="${COMPOSE_PROJECT_NAME:-subcult-os}"
api_url="${API_URL:-}"

if [[ "${paid_mode}" == true ]]; then
  if [[ -z "${STRIPE_SECRET_KEY:-}" || -z "${STRIPE_WEBHOOK_SECRET:-}" ]]; then
    echo "Skipping paid alpha QA: set STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET to run Stripe checks."
    exit 0
  fi
fi

if [[ -z "${api_url}" ]]; then
  api_port="$(docker compose -p "${project_name}" port api 8080 | awk -F: '{print $NF}')"
  if [[ -z "${api_port}" ]]; then
    echo "Could not determine API port. Is the stack running? Try: make up-build" >&2
    exit 1
  fi
  api_url="http://localhost:${api_port}"
fi

tmpdir="$(mktemp -d /tmp/subcult-alpha-qa.XXXXXX)"
trap 'rm -rf "${tmpdir}"' EXIT

owner_cookie="${tmpdir}/owner.cookies"
member_cookie="${tmpdir}/member.cookies"
suffix="$(date +%s%N)"
owner="owner+${suffix}@example.test"
member="member+${suffix}@example.test"
guest="guest+${suffix}@example.test"

json_get() {
  python -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$1"
}

echo "Alpha QA against ${api_url}"

curl -fsS -c "${owner_cookie}" -H 'Content-Type: application/json' \
  -d "{\"email\":\"${owner}\",\"password\":\"secret1234\",\"displayName\":\"Owner\"}" \
  "${api_url}/api/auth/signup" >/dev/null

workspace_json="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
  -d '{"name":"Alpha QA Collective"}' \
  "${api_url}/api/workspaces")"
workspace_id="$(json_get id <<<"${workspace_json}")"

invite_json="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
  -d "{\"email\":\"${member}\"}" \
  "${api_url}/api/workspaces/${workspace_id}/invitations")"
invite_token="$(json_get token <<<"${invite_json}")"

outbox_json="$(curl -fsS -b "${owner_cookie}" "${api_url}/api/dev/email-outbox")"
INVITE_TOKEN="${invite_token}" python -c 'import json,os,sys; data=json.load(sys.stdin); token=os.environ["INVITE_TOKEN"]; assert any(token in msg.get("body", "") for msg in data), data; print("✓ dev outbox contains invite link")' <<<"${outbox_json}"

curl -fsS -c "${member_cookie}" -H 'Content-Type: application/json' \
  -d "{\"email\":\"${member}\",\"password\":\"secret1234\",\"displayName\":\"Door\"}" \
  "${api_url}/api/auth/signup" >/dev/null

curl -fsS -b "${member_cookie}" -H 'Content-Type: application/json' \
  -d '{}' \
  "${api_url}/api/invitations/${invite_token}/accept" >/dev/null

event_json="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
  -d '{"title":"Alpha QA Night","startsAt":"2026-07-01T20:00:00Z","publicDescription":"Free community event.","locationDisplay":"Warehouse District","ticketAllocation":1}' \
  "${api_url}/api/workspaces/${workspace_id}/events")"
event_id="$(json_get id <<<"${event_json}")"

published_json="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
  -d '{}' \
  "${api_url}/api/events/${event_id}/publish")"
slug="$(json_get publicSlug <<<"${published_json}")"

ticket_json="$(curl -fsS -H 'Content-Type: application/json' \
  -d "{\"email\":\"${guest}\",\"displayName\":\"Guest\"}" \
  "${api_url}/api/public/events/${slug}/reservations")"
code="$(json_get code <<<"${ticket_json}")"

second_status="$(curl -s -o "${tmpdir}/full.json" -w '%{http_code}' -H 'Content-Type: application/json' \
  -d "{\"email\":\"second-${guest}\"}" \
  "${api_url}/api/public/events/${slug}/reservations")"
if [[ "${second_status}" != "409" ]]; then
  echo "Expected full reservation to return 409, got ${second_status}" >&2
  exit 1
fi
echo "✓ full capacity returns conflict"

search_json="$(curl -fsS -b "${member_cookie}" "${api_url}/api/events/${event_id}/door/tickets?query=${code}")"
python -c 'import json,sys; data=json.load(sys.stdin); assert len(data) == 1, data; print("✓ exact-code door search returns one ticket")' <<<"${search_json}"

curl -fsS -b "${member_cookie}" -H 'Content-Type: application/json' \
  -d "{\"code\":\"${code}\"}" \
  "${api_url}/api/events/${event_id}/door/check-ins" >/dev/null

curl -fsS -b "${member_cookie}" -H 'Content-Type: application/json' \
  -d "{\"code\":\"${code}\"}" \
  "${api_url}/api/events/${event_id}/door/check-ins" >/dev/null
echo "✓ duplicate check-in is idempotent"

report_json="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
  -d '{}' \
  "${api_url}/api/events/${event_id}/end-of-night")"
python -c 'import json,sys; r=json.load(sys.stdin); assert r["ticketsReserved"] == 1 and r["ticketsCheckedIn"] == 1 and r["noShows"] == 0, r; print("✓ end-of-night report counts are correct")' <<<"${report_json}"

if [[ "${paid_mode}" != true ]]; then
  echo "Alpha QA passed."
  exit 0
fi

echo "Free alpha QA passed."
echo "Running paid alpha QA..."

paid_event_json="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
  -d '{"title":"Alpha QA Paid Night","startsAt":"2026-07-02T20:00:00Z","publicDescription":"Paid community event.","locationDisplay":"Warehouse District","ticketAllocation":2,"pricingMode":"fixed","ticketPriceCents":1500,"ticketCurrency":"usd"}' \
  "${api_url}/api/workspaces/${workspace_id}/events")"
paid_event_id="$(json_get id <<<"${paid_event_json}")"

paid_published_json="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
  -d '{}' \
  "${api_url}/api/events/${paid_event_id}/publish")"
paid_slug="$(json_get publicSlug <<<"${paid_published_json}")"

paid_free_status="$(curl -sS -o "${tmpdir}/paid-free.json" -w '%{http_code}' -H 'Content-Type: application/json' \
  -d "{\"email\":\"${guest}\",\"displayName\":\"Guest\"}" \
  "${api_url}/api/public/events/${paid_slug}/reservations")"
if [[ "${paid_free_status}" != "409" ]]; then
  echo "Expected free reservation on paid event to return 409, got ${paid_free_status}" >&2
  cat "${tmpdir}/paid-free.json" >&2 || true
  exit 1
fi
echo "✓ paid event rejects free reservation path"

paid_reservation_body="{\"email\":\"paid-${guest}\",\"displayName\":\"Guest\"}"
paid_reservation_status="$(curl -sS -o "${tmpdir}/paid-reservation.json" -w '%{http_code}' -H 'Content-Type: application/json' \
  -d "${paid_reservation_body}" \
  "${api_url}/api/public/events/${paid_slug}/paid-reservations")"
if [[ "${paid_reservation_status}" == "503" ]]; then
  echo "Payment provider unavailable while STRIPE_SECRET_KEY/STRIPE_WEBHOOK_SECRET are set; paid QA cannot continue." >&2
  cat "${tmpdir}/paid-reservation.json" >&2 || true
  exit 1
fi
if [[ "${paid_reservation_status}" != "200" ]]; then
  echo "Expected paid reservation to return 200, got ${paid_reservation_status}" >&2
  cat "${tmpdir}/paid-reservation.json" >&2 || true
  exit 1
fi

checkout_url="$(python -c 'import json,sys; data=json.load(sys.stdin); print(data.get("checkoutUrl", ""))' < "${tmpdir}/paid-reservation.json")"
if [[ -z "${checkout_url}" ]]; then
  echo "Paid reservation response did not include a checkoutUrl." >&2
  cat "${tmpdir}/paid-reservation.json" >&2 || true
  exit 1
fi
echo "✓ paid reservation returns checkoutUrl"
echo "Paid alpha QA passed."
