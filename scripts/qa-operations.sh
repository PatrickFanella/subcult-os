#!/usr/bin/env bash
# Operations-panel rehearsal: contacts, commitments, staffing, templates,
# roles/applications, reminder boundaries, and workspace invitations/switching.
# Same conventions as scripts/alpha-qa.sh: synthetic verified accounts only,
# a disposable target enforced by qa-identity.sh, and no secrets printed.
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

tmpdir="$(mktemp -d /tmp/subcult-operations-qa.XXXXXX)"
trap 'rm -rf "${tmpdir}"' EXIT

owner_cookie="${tmpdir}/owner.cookies"
member_cookie="${tmpdir}/member.cookies"
suffix="$(date +%s%N)"
owner_email="owner+${suffix}@example.test"
member_email="member+${suffix}@example.test"
applicant_email="applicant+${suffix}@example.test"

pass_count=0
fail_count=0
declare -a results=()

json_get() {
  python -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$1"
}

step() {
  local name="$1"
  shift
  if "$@"; then
    echo "PASS: ${name}"
    results+=("PASS ${name}")
    pass_count=$((pass_count + 1))
  else
    echo "FAIL: ${name}" >&2
    results+=("FAIL ${name}")
    fail_count=$((fail_count + 1))
  fi
}

req_status() {
  # req_status <out-file> <curl args...> ; prints the http status code
  local out="$1"
  shift
  curl -sS -o "${out}" -w '%{http_code}' "$@"
}

echo "Operations QA against ${api_url}"

# --- Accounts ---------------------------------------------------------

signup_and_verify() {
  local cookie="$1" email="$2" name="$3"
  curl -fsS -c "${cookie}" -H 'Content-Type: application/json' \
    -d "{\"email\":\"${email}\",\"password\":\"secret1234\",\"displayName\":\"${name}\"}" \
    "${api_url}/api/auth/signup" >/dev/null
  qa_verify_signup "${cookie}" "${email}"
}

step "create owner account" signup_and_verify "${owner_cookie}" "${owner_email}" "Owner"
step "create member account" signup_and_verify "${member_cookie}" "${member_email}" "Member"

# --- Workspace, invitation, switching -----------------------------------

create_workspace() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"name":"Operations QA Collective"}' \
    "${api_url}/api/workspaces")"
  workspace_id="$(json_get id <<<"${body}")"
  [[ -n "${workspace_id}" ]]
}
step "owner creates workspace" create_workspace

invite_member() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d "{\"email\":\"${member_email}\"}" \
    "${api_url}/api/workspaces/${workspace_id}/invitations")"
  invite_token="$(json_get token <<<"${body}")"
  [[ -n "${invite_token}" ]]
}
step "owner invites member" invite_member

accept_invitation() {
  curl -fsS -b "${member_cookie}" -H 'Content-Type: application/json' \
    -d '{}' \
    "${api_url}/api/invitations/${invite_token}/accept" >/dev/null
}
step "member accepts invitation" accept_invitation

member_id=""
fetch_member_person_id() {
  local body
  body="$(curl -fsS -b "${member_cookie}" "${api_url}/api/me")"
  member_id="$(json_get id <<<"${body}")"
  [[ -n "${member_id}" ]]
}
step "resolve member personId via /api/me" fetch_member_person_id

switch_workspace_as_member() {
  local body role
  body="$(curl -fsS -b "${member_cookie}" "${api_url}/api/workspaces/${workspace_id}")"
  role="$(json_get role <<<"${body}")"
  [[ "${role}" == "member" ]]
}
step "member switches into invited workspace" switch_workspace_as_member

owner_current_workspace() {
  local body id
  body="$(curl -fsS -b "${owner_cookie}" "${api_url}/api/workspaces/current")"
  id="$(json_get id <<<"${body}")"
  [[ "${id}" == "${workspace_id}" ]]
}
step "owner's current workspace resolves to the created workspace" owner_current_workspace

# --- Event -------------------------------------------------------------

event_id=""
create_event() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"title":"Operations QA Night","startsAt":"2026-08-01T20:00:00Z","publicDescription":"Operations rehearsal.","locationDisplay":"Warehouse District","ticketAllocation":10}' \
    "${api_url}/api/workspaces/${workspace_id}/events")"
  event_id="$(json_get id <<<"${body}")"
  [[ -n "${event_id}" ]]
}
step "owner creates draft event" create_event

# --- Contacts ------------------------------------------------------------

contact_id=""
create_contact() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"displayName":"Fixture Contact","email":"fixture-contact@example.test","phone":"555-0100","notes":"Sound tech.","tags":["sound","vendor"]}' \
    "${api_url}/api/workspaces/${workspace_id}/contacts")"
  contact_id="$(json_get id <<<"${body}")"
  [[ -n "${contact_id}" ]]
}
step "create contact" create_contact

list_contacts() {
  local body count
  body="$(curl -fsS -b "${owner_cookie}" "${api_url}/api/workspaces/${workspace_id}/contacts")"
  count="$(python -c 'import json,sys; print(len(json.load(sys.stdin)))' <<<"${body}")"
  [[ "${count}" == "1" ]]
}
step "list contacts returns the created contact" list_contacts

update_contact() {
  local body name
  body="$(curl -fsS -X PATCH -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"displayName":"Fixture Contact (Updated)","tags":["sound"]}' \
    "${api_url}/api/workspaces/${workspace_id}/contacts/${contact_id}")"
  name="$(json_get displayName <<<"${body}")"
  [[ "${name}" == "Fixture Contact (Updated)" ]]
}
step "update contact" update_contact


# --- Commitments ---------------------------------------------------------

commitment_id=""
create_commitment() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d "{\"title\":\"Confirm vendor load-in\",\"description\":\"Call vendor to confirm timing.\",\"eventId\":\"${event_id}\",\"contactId\":\"${contact_id}\"}" \
    "${api_url}/api/workspaces/${workspace_id}/commitments")"
  commitment_id="$(json_get id <<<"${body}")"
  [[ -n "${commitment_id}" ]]
}
step "create commitment linked to event and contact" create_commitment

reject_invalid_commitment_status() {
  local status
  status="$(req_status "${tmpdir}/commitment-bad-status.json" -X PATCH -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"status":"not-a-real-status"}' \
    "${api_url}/api/workspaces/${workspace_id}/commitments/${commitment_id}")"
  [[ "${status}" == "400" ]]
}
step "commitment update rejects an invalid status" reject_invalid_commitment_status

complete_commitment() {
  local body status completed_at
  body="$(curl -fsS -X PATCH -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"status":"done"}' \
    "${api_url}/api/workspaces/${workspace_id}/commitments/${commitment_id}")"
  status="$(json_get status <<<"${body}")"
  completed_at="$(json_get completedAt <<<"${body}")"
  [[ "${status}" == "done" && -n "${completed_at}" && "${completed_at}" != "None" ]]
}
step "mark commitment done sets completedAt" complete_commitment

# A second commitment stays open with a past due date, to drive the reminder sweep below.
overdue_commitment_id=""
create_overdue_commitment() {
  local due body
  due="$(python -c 'from datetime import datetime, timedelta, timezone; print((datetime.now(timezone.utc) - timedelta(hours=1)).replace(microsecond=0).isoformat().replace("+00:00","Z"))')"
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d "{\"title\":\"Overdue vendor payment\",\"description\":\"Pay vendor deposit.\",\"eventId\":\"${event_id}\",\"dueAt\":\"${due}\"}" \
    "${api_url}/api/workspaces/${workspace_id}/commitments")"
  overdue_commitment_id="$(json_get id <<<"${body}")"
  [[ -n "${overdue_commitment_id}" ]]
}
step "create overdue open commitment for reminder sweep" create_overdue_commitment

# A commitment due far in the future must not produce a reminder.
create_future_commitment() {
  local due body id
  due="$(python -c 'from datetime import datetime, timedelta, timezone; print((datetime.now(timezone.utc) + timedelta(days=30)).replace(microsecond=0).isoformat().replace("+00:00","Z"))')"
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d "{\"title\":\"Future vendor payment\",\"description\":\"Pay vendor final balance.\",\"eventId\":\"${event_id}\",\"dueAt\":\"${due}\"}" \
    "${api_url}/api/workspaces/${workspace_id}/commitments")"
  id="$(json_get id <<<"${body}")"
  [[ -n "${id}" ]]
}
step "create future open commitment that must not trigger a reminder" create_future_commitment

# --- Staffing --------------------------------------------------------------

staffing_id=""
member_cannot_create_staffing() {
  local status
  status="$(req_status "${tmpdir}/staffing-forbidden.json" -b "${member_cookie}" -H 'Content-Type: application/json' \
    -d '{"title":"Open doors","kind":"task","notes":"","startsAt":null,"endsAt":null}' \
    "${api_url}/api/events/${event_id}/staffing")"
  [[ "${status}" == "403" ]]
}
step "member cannot create staffing items (owner-only boundary)" member_cannot_create_staffing

create_staffing() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"title":"Run door","kind":"shift","notes":"Scan tickets at entrance.","startsAt":null,"endsAt":null}' \
    "${api_url}/api/events/${event_id}/staffing")"
  staffing_id="$(json_get id <<<"${body}")"
  [[ -n "${staffing_id}" ]]
}
step "owner creates staffing item" create_staffing

assign_staffing_to_member() {
  local body assigned status
  body="$(curl -fsS -X PATCH -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d "{\"assignedPersonId\":\"${member_id}\",\"status\":\"assigned\"}" \
    "${api_url}/api/events/${event_id}/staffing/${staffing_id}")"
  assigned="$(json_get assignedPersonId <<<"${body}")"
  status="$(json_get status <<<"${body}")"
  [[ "${assigned}" == "${member_id}" && "${status}" == "assigned" ]]
}
step "owner assigns staffing item to member" assign_staffing_to_member

member_cannot_patch_staffing() {
  local status
  status="$(req_status "${tmpdir}/staffing-patch-forbidden.json" -X PATCH -b "${member_cookie}" -H 'Content-Type: application/json' \
    -d '{"status":"completed"}' \
    "${api_url}/api/events/${event_id}/staffing/${staffing_id}")"
  [[ "${status}" == "403" ]]
}
step "member cannot PATCH staffing items (owner-only boundary)" member_cannot_patch_staffing

# --- Templates ---------------------------------------------------------

template_id=""
create_template() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"name":"Standard warehouse night","title":"Warehouse Night","publicDescription":"Recurring warehouse show.","locationDisplay":"Warehouse District","ticketAllocation":40,"pricingMode":"free","ticketPriceCents":0,"ticketCurrency":"usd","privateNotes":"Bring backup generator."}' \
    "${api_url}/api/workspaces/${workspace_id}/event-templates")"
  template_id="$(json_get id <<<"${body}")"
  [[ -n "${template_id}" ]]
}
step "create event template" create_template

templated_event_id=""
create_draft_for_template() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"title":"Placeholder","startsAt":"2026-09-01T20:00:00Z","publicDescription":"placeholder","locationDisplay":"placeholder","ticketAllocation":1}' \
    "${api_url}/api/workspaces/${workspace_id}/events")"
  templated_event_id="$(json_get id <<<"${body}")"
  [[ -n "${templated_event_id}" ]]
}
step "create draft event to receive the template" create_draft_for_template

apply_template() {
  local body title allocation
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d "{\"templateId\":\"${template_id}\"}" \
    "${api_url}/api/events/${templated_event_id}/apply-template")"
  title="$(json_get title <<<"${body}")"
  allocation="$(json_get ticketAllocation <<<"${body}")"
  [[ "${title}" == "Warehouse Night" && "${allocation}" == "40" ]]
}
step "apply template to draft event" apply_template

apply_template_to_published_event_conflicts() {
  local publish_status apply_status
  publish_status="$(req_status "${tmpdir}/publish-for-template.json" -b "${owner_cookie}" -H 'Content-Type: application/json' -d '{}' \
    "${api_url}/api/events/${templated_event_id}/publish")"
  [[ "${publish_status}" == "200" ]] || return 1
  apply_status="$(req_status "${tmpdir}/apply-template-conflict.json" -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d "{\"templateId\":\"${template_id}\"}" \
    "${api_url}/api/events/${templated_event_id}/apply-template")"
  [[ "${apply_status}" == "409" ]]
}
step "template cannot be applied once the event is published" apply_template_to_published_event_conflicts

# --- Roles and applications ---------------------------------------------

public_role_id=""
create_public_role() {
  local body
  body="$(curl -fsS -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"name":"Door volunteer","description":"Applicant-facing role.","capacity":2,"public":true}' \
    "${api_url}/api/events/${event_id}/roles")"
  public_role_id="$(json_get id <<<"${body}")"
  [[ -n "${public_role_id}" ]]
}
step "create public role" create_public_role

publish_event_for_applications() {
  local status
  status="$(req_status "${tmpdir}/publish-main.json" -b "${owner_cookie}" -H 'Content-Type: application/json' -d '{}' \
    "${api_url}/api/events/${event_id}/publish")"
  [[ "${status}" == "200" ]]
}
step "publish event so public roles/applications are reachable" publish_event_for_applications

application_id=""
submit_application() {
  local body event_slug
  event_slug="$(json_get publicSlug < "${tmpdir}/publish-main.json")"
  body="$(curl -fsS -H 'Content-Type: application/json' \
    -d "{\"roleId\":\"${public_role_id}\",\"applicantName\":\"Fixture Applicant\",\"applicantEmail\":\"${applicant_email}\",\"message\":\"Happy to help.\"}" \
    "${api_url}/api/public/events/${event_slug}/role-applications")"
  application_id="$(json_get id <<<"${body}")"
  [[ -n "${application_id}" ]]
}
step "public applicant submits role application" submit_application

reject_invalid_review_status() {
  local status
  status="$(req_status "${tmpdir}/review-bad-status.json" -X PATCH -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"status":"not-a-real-status"}' \
    "${api_url}/api/events/${event_id}/role-applications/${application_id}")"
  [[ "${status}" == "400" ]]
}
step "role application review rejects an invalid status" reject_invalid_review_status

decide_application() {
  local body status
  body="$(curl -fsS -X PATCH -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{"status":"accepted"}' \
    "${api_url}/api/events/${event_id}/role-applications/${application_id}")"
  status="$(json_get status <<<"${body}")"
  [[ "${status}" == "accepted" ]]
}
step "owner accepts role application" decide_application

member_cannot_review_application() {
  # Re-submit a second application, then confirm member cannot decide it (owner-only boundary).
  local body second_application_id status event_slug
  event_slug="$(json_get publicSlug < "${tmpdir}/publish-main.json")"
  body="$(curl -fsS -H 'Content-Type: application/json' \
    -d "{\"roleId\":\"${public_role_id}\",\"applicantName\":\"Second Applicant\",\"applicantEmail\":\"second-${applicant_email}\",\"message\":\"Also available.\"}" \
    "${api_url}/api/public/events/${event_slug}/role-applications")"
  second_application_id="$(json_get id <<<"${body}")"
  status="$(req_status "${tmpdir}/review-forbidden.json" -X PATCH -b "${member_cookie}" -H 'Content-Type: application/json' \
    -d '{"status":"accepted"}' \
    "${api_url}/api/events/${event_id}/role-applications/${second_application_id}")"
  [[ "${status}" == "403" ]]
}
step "member cannot decide role applications (owner-only boundary)" member_cannot_review_application

# --- Reminder boundaries -------------------------------------------------

member_cannot_sweep_reminders() {
  local status
  status="$(req_status "${tmpdir}/sweep-forbidden.json" -X POST -b "${member_cookie}" -H 'Content-Type: application/json' \
    -d '{}' \
    "${api_url}/api/workspaces/${workspace_id}/reminders/sweep")"
  [[ "${status}" == "403" ]]
}
step "member cannot sweep reminders (owner-only boundary)" member_cannot_sweep_reminders

sweep_before_due_state_produces_no_reminder_for_future_item() {
  # A prior sweep must not have created a reminder for the far-future commitment.
  local body count
  body="$(curl -fsS -b "${owner_cookie}" "${api_url}/api/workspaces/${workspace_id}/reminders")"
  count="$(python -c 'import json,sys; data=json.load(sys.stdin); print(len(data))' <<<"${body}")"
  [[ "${count}" == "0" ]]
}
step "no reminders exist before the overdue commitment is swept" sweep_before_due_state_produces_no_reminder_for_future_item

sweep_reminders_first_pass() {
  local body created
  body="$(curl -fsS -X POST -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{}' \
    "${api_url}/api/workspaces/${workspace_id}/reminders/sweep")"
  created="$(json_get createdCount <<<"${body}")"
  [[ "${created}" == "1" ]]
}
step "owner sweeps reminders and creates exactly one for the overdue commitment" sweep_reminders_first_pass

reminder_list_reflects_overdue_commitment_only() {
  local body count source_id
  body="$(curl -fsS -b "${owner_cookie}" "${api_url}/api/workspaces/${workspace_id}/reminders")"
  count="$(python -c 'import json,sys; print(len(json.load(sys.stdin)))' <<<"${body}")"
  source_id="$(python -c 'import json,sys; print(json.load(sys.stdin)[0]["sourceId"])' <<<"${body}")"
  [[ "${count}" == "1" && "${source_id}" == "${overdue_commitment_id}" ]]
}
step "reminder list contains only the overdue commitment's reminder" reminder_list_reflects_overdue_commitment_only

sweep_reminders_second_pass_is_idempotent() {
  local body created
  body="$(curl -fsS -X POST -b "${owner_cookie}" -H 'Content-Type: application/json' \
    -d '{}' \
    "${api_url}/api/workspaces/${workspace_id}/reminders/sweep")"
  created="$(json_get createdCount <<<"${body}")"
  [[ "${created}" == "0" ]]
}
step "repeat sweep does not resend the same reminder (idempotency boundary)" sweep_reminders_second_pass_is_idempotent

# --- Summary -------------------------------------------------------------

echo
echo "Operations QA summary:"
for line in "${results[@]}"; do
  echo "  ${line}"
done
echo "${pass_count} passed, ${fail_count} failed."

if ((fail_count > 0)); then
  exit 1
fi
echo "Operations QA passed."
