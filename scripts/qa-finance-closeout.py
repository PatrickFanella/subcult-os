"""API helper for qa-finance-closeout.sh; cookie files and DTOs stay private."""

import csv
import datetime
import http.cookiejar
import io
import json
from pathlib import Path
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
import uuid


class RehearsalError(Exception):
    pass


def require(condition, message):
    if not condition:
        raise RehearsalError(message)


def main():
    require(len(sys.argv) == 5, "Run the qa-finance-closeout.sh wrapper.")
    api_url, owner_cookie, crew_cookie, crew_email = sys.argv[1:]
    # The helper also guards direct invocation before any network request.
    guard = Path(__file__).with_name("qa-identity.sh")
    subprocess.run(
        ["bash", "-c", 'source "$1"; api_url="$2"; qa_require_disposable_target',
         "finance-qa-guard", str(guard), api_url], check=True,
    )
    require(crew_email.endswith("@example.test"), "Synthetic crew email required.")
    api_url = api_url.rstrip("/")
    passed = 0

    def check(condition, label):
        nonlocal passed
        require(condition, label)
        passed += 1
        print(f"PASS: {label}", flush=True)

    def client(cookie=None):
        jar = http.cookiejar.MozillaCookieJar(cookie)
        if cookie:
            jar.load(ignore_discard=True)
            # urllib uses localhost.local as the effective single-label host;
            # curl stores host-only localhost cookies under localhost instead.
            if urllib.parse.urlsplit(api_url).hostname == "localhost":
                for cookie in list(jar):
                    if cookie.domain == "localhost" and not cookie.domain_specified:
                        jar.clear(cookie.domain, cookie.path, cookie.name)
                        cookie.domain = "localhost.local"
                        jar.set_cookie(cookie)
        return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))

    owner, crew, anonymous = client(owner_cookie), client(crew_cookie), client()

    def request(actor, path, payload=None, method=None):
        data = None if payload is None else json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(api_url + path, data=data, method=method,
                                     headers={"Content-Type": "application/json"})
        try:
            response = actor.open(req, timeout=20)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            return response.status, response.headers, response.read().decode("utf-8")

    def read(actor, path, payload=None, method=None, status=200):
        code, headers, body = request(actor, path, payload, method)
        require(code == status, f"API status mismatch: expected {status}, received {code}.")
        try:
            return json.loads(body), headers
        except ValueError:
            raise RehearsalError("API returned invalid JSON.") from None

    workspace, _ = read(owner, "/api/workspaces", {"name": "Finance closeout rehearsal"})
    base = "/api/workspaces/" + workspace["id"]
    invitation, _ = read(owner, base + "/invitations", {"email": crew_email})
    read(crew, "/api/invitations/" + invitation["token"] + "/accept", {})
    roster, _ = read(owner, base)
    membership = next(row for row in roster["members"] if row["email"] == crew_email)
    member_path = base + "/members/" + membership["id"]
    check(membership["accessState"] == "active", "fresh verified crew membership")

    starts = (datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(days=10)).replace(microsecond=0)
    event, _ = read(owner, base + "/events", {
        "title": "Synthetic finance and reuse night", "startsAt": starts.isoformat(),
        "publicDescription": "Free synthetic rehearsal.", "locationDisplay": "Synthetic room",
        "ticketAllocation": 2, "pricingMode": "free", "ticketPriceCents": 0, "ticketCurrency": "usd",
    })
    path = "/api/events/" + event["id"]
    sentinel = "PRIVATE_FINANCE_" + uuid.uuid4().hex
    read(crew, path + "/finance-lines", status=403)
    check(True, "baseline crew cannot read finance")

    def record(entry_type, amount, **extra):
        payload = {"entryType": entry_type, "direction": "expense", "currency": "usd",
                   "amountCents": amount, "label": "Synthetic " + entry_type,
                   "reason": sentinel, "requestKey": str(uuid.uuid4()), **extra}
        value, _ = read(owner, path + "/finance-lines", payload)
        require(value["eventId"] == event["id"], "Finance receipt event mismatch.")
        return payload, value

    record("budget", 12555)
    _, payable = record("payable", 8033)
    _, actual = record("actual_payment", 3001, occurredAt="2026-01-02T03:04:05-05:00", payableLineId=payable["id"])
    correction_payload, correction = record("actual_payment", 2501,
        occurredAt="2026-01-02T03:04:05-05:00", payableLineId=payable["id"], correctsLineId=actual["id"])
    check(correction["occurredAt"] == "2026-01-02T08:04:05Z", "manual occurrence normalized to UTC")
    replay, _ = read(owner, path + "/finance-lines", correction_payload)
    check(replay["id"] == correction["id"], "same request key replays one correction")
    read(owner, path + "/finance-lines", {**correction_payload, "amountCents": 2000}, status=409)
    check(True, "changed payload with same key rejected")
    lines, _ = read(owner, path + "/finance-lines")
    corrected_ids = {line["correctsLineId"] for line in lines if line.get("correctsLineId")}
    current = [line for line in lines if line["id"] not in corrected_ids]
    expected = {"budget": 12555, "payable": 8033, "actual_payment": 2501}
    totals = {kind: sum(line["amountCents"] for line in current if line["entryType"] == kind) for kind in expected}
    check(len(lines) == 4 and totals == expected, "retained history and separate current categories")
    check(correction["payableLineId"] == payable["id"], "correction preserves stable payable root")

    read(owner, member_path, {"role": "finance"}, "PATCH")
    read(crew, path + "/finance-lines")
    check(True, "assigned Finance can read ledger")
    read(owner, member_path, {"role": "crew"}, "PATCH")
    read(crew, path + "/finance-lines", status=403)
    read(crew, path + "/finance-lines", correction_payload, status=403)
    check(True, "removing Finance denies reads and writes")

    published, _ = read(owner, path + "/publish", {})
    ticket, _ = read(anonymous, "/api/public/events/" + published["publicSlug"] + "/reservations",
                     {"email": "guest+" + uuid.uuid4().hex + "@example.test", "displayName": "Synthetic guest"})
    read(owner, path + "/door/check-ins", {"code": ticket["code"]})
    report, _ = read(owner, path + "/end-of-night", {})
    check((report["ticketsReserved"], report["ticketsCheckedIn"], report["noShows"]) == (1, 1, 0), "free door and closeout counts")
    settlement, _ = read(owner, path + "/settlement")
    check(settlement["grossPaidRevenueCents"] == 0 and settlement["freeTicketCount"] == 1, "manual actuals do not become paid ticket revenue")
    finalized, _ = read(owner, path + "/settlement/finalize", {})
    check(finalized["status"] == "finalized" and finalized["grossPaidRevenueCents"] == 0, "free settlement finalized")
    read(owner, path + "/settlement/adjustments", {"amountCents": 100, "label": "Late", "reason": "Rejected"}, status=409)
    check(True, "finalized settlement rejects later adjustment")

    archive, _ = read(owner, path + "/archive")
    read(anonymous, path + "/archive", status=401)
    check(archive["status"] == "private", "archive stays private and anonymous access denied")
    archive_sentinel = "PRIVATE_ARCHIVE_" + uuid.uuid4().hex
    read(owner, path + "/archive/notes", {"body": archive_sentinel})
    archive, _ = read(owner, path + "/archive")
    check(archive["noteCount"] == 1 and archive["notes"][0]["body"] == archive_sentinel, "private lesson retained in archive")

    seed, _ = read(owner, path + "/archive/seed-draft", {})
    again, _ = read(owner, path + "/archive/seed-draft", {})
    check(seed["id"] != event["id"] and again["id"] == seed["id"], "archive seed retry returns one distinct draft")
    seed_path = "/api/events/" + seed["id"]
    check(seed["status"] == "draft" and seed["publicSlug"] is None
          and seed["reservedCount"] == 0 and seed["checkedInCount"] == 0, "seed has no publication or ticket state")
    for collection in ["finance-lines", "roles", "staffing"]:
        value, _ = read(owner, seed_path + "/" + collection)
        check(value == [], "seed has no copied " + collection)
    read(owner, seed_path + "/archive", status=404)
    check(True, "seed has no copied private archive")

    template_sentinel = "PRIVATE_TEMPLATE_" + uuid.uuid4().hex
    template, _ = read(owner, base + "/event-templates", {
        "name": "Synthetic reuse template", "title": "Reused synthetic night",
        "publicDescription": "Public synthetic description.", "locationDisplay": "Synthetic room",
        "ticketAllocation": 2, "pricingMode": "free", "ticketPriceCents": 0,
        "ticketCurrency": "usd", "privateNotes": template_sentinel,
    })
    applied, _ = read(owner, seed_path + "/apply-template", {"templateId": template["id"]})
    check(applied["status"] == "draft" and applied["title"] == template["title"], "template applies to draft")
    republished, _ = read(owner, seed_path + "/publish", {})
    read(owner, seed_path + "/apply-template", {"templateId": template["id"]}, status=409)
    check(True, "published event rejects template application")
    public, _ = read(anonymous, "/api/public/events/" + republished["publicSlug"])
    public_text = json.dumps(public)
    check(all(value not in public_text for value in [sentinel, archive_sentinel, template_sentinel,
          "privateNotes", "financeLines", "settlementId", "archiveId"]), "public event excludes private record and note fields")

    for suffix in ["settlement.csv", "settlement.md", "settlement-print.html"]:
        export_path = path + "/exports/" + suffix
        code, headers, text = request(owner, export_path)
        require(code == 200 and headers.get("Cache-Control") == "no-store", "Private export status/cache mismatch.")
        require(request(crew, export_path)[0] == 403 and request(anonymous, export_path)[0] == 401, "Export authority mismatch.")
        require(archive_sentinel not in text and template_sentinel not in text, "Unrelated private note in export.")
        if suffix.endswith(".csv"):
            rows = list(csv.DictReader(io.StringIO(text)))
            history = [row for row in rows if row["record_type"] == "finance_line_history"]
            exported = {row["finance_entry_type"]: int(row["finance_current_total_cents"])
                        for row in rows if row["record_type"] == "finance_current_total"}
            require(len(history) == 4 and exported == expected, "CSV history/current totals mismatch.")
            require(sum(row["finance_is_current"] == "" for row in history) == 1, "CSV superseded actual mismatch.")
            require(any(row["finance_occurred_at"] == "2026-01-02T08:04:05Z" for row in history), "CSV UTC timestamp mismatch.")
        else:
            require(all(value in text for value in ["Budgets", "Recorded obligations", "Manual recorded payments",
                    "125.55", "80.33", "25.01"]), "Report categories/exact cents mismatch.")
        check(True, suffix + " scope, UTF-8, history/totals and unrelated-note isolation")

    print(f"Finance closeout API rehearsal passed: {passed} checks. Fresh synthetic records retained; no provider call."
          " Browser/device/Print-PDF and accounting-user qualification remain separate.")


if __name__ == "__main__":
    try:
        main()
    except (RehearsalError, subprocess.CalledProcessError, urllib.error.URLError,
            ValueError, KeyError, TypeError, StopIteration, OSError) as error:
        # No DTO, cookie, invitation token, ticket capability or URL in failure output.
        print("Finance closeout rehearsal failed: " + (str(error).rstrip(".") if isinstance(error, RehearsalError)
              else type(error).__name__) + ". Synthetic evidence is retained; inspect the owned target.", file=sys.stderr)
        sys.exit(1)
