# One scoped announcement channel and delivery worker (SIGNAL-01, Issue #24)

Status: one channel decision, migration 000014, draft/preview/schedule/cancel/list/get
endpoints, dispatch, and a synthetic grant/schedule/withdraw/dispatch/send
journey implemented 2026-09-24. This is the future feature
[`consent.md`](consent.md) named as not yet built: the first (and, per the
decision below, only) code path that enqueues an
`email_outbox` row with `purpose = 'announcement'`.

## Channel decision

**Verified email through the existing Resend outbox is the one channel
this slice implements.** No SMS provider, push channel or social DM
automation is added.

Rationale: email is the only channel in this codebase that already has,
qualified and tested, every piece an announcement send needs — a consent
grant type (`consent_grants`, CONSENT-01), a delivery ledger with bounded
retries and terminal states (`email_outbox`, transactional-email.md),
suppression (`email_suppressions`), and a provider adapter
(`internal/mail`, Resend). Adding SMS would mean a second consent flow
(a new `channel` value, a new verification/withdrawal credential) and a
second per-message cost basis, for a channel with no observed pilot
demand yet. Building a second channel before the first one has completed
even a limited permissioned pilot (see "Pilot gate" below) would be
speculative work against unverified demand — the same reasoning
`consent.md`'s "What this is not" section applies to marketing sends in
general.

## Cost basis

`ANNOUNCEMENT_UNIT_COST_CENTS` (default `0`) is a per-recipient cost
estimate used only to compute `estimated_cost_cents` (recipient count ×
unit cost) for preview and reporting. It is observed operator-supplied
configuration, not a number this slice invents: the default `0` means
"cost unknown," not "free." It never gates whether an announcement
dispatches, and it is never sent to the provider.

## Model (migration 000014)

- **`announcements`** — one row per drafted, scheduled or dispatched
  announcement. `status` is forward-only: `draft` → `scheduled` →
  `dispatching` → `dispatched`, or `draft`/`scheduled` → `cancelled`.
  `recipient_count`, `withheld_count` and `estimated_cost_cents` are all
  zero until dispatch actually runs; they are never estimated or
  persisted at schedule time, because the real audience is only known at
  dispatch (see "Dispatch" below).
- **`announcement_deliveries`** — links an announcement to each
  `email_outbox` row it produced (`announcement_id`, `outbox_id`,
  `recipient_address`). It carries no delivery outcome, retry count or
  suppression state of its own: those are read from
  `email_outbox.delivery_status` by joining through `outbox_id`, so
  provider truth is never duplicated. A unique index on `outbox_id`
  reflects that dispatch enqueues exactly one outbox row per allowed
  recipient.
- **`consent_grants.withdraw_token_hash`** (nullable) — see "Withdraw
  link" below.

## Withdraw link

Every announcement email must carry the recipient's withdraw link — the
same tokenized `/consent/withdraw` page CONSENT-01 added. The problem:
`verification_token_hash`'s raw token is a one-time secret, sent once in
the confirm/unsubscribe email and never retained, so it cannot be looked
up or reconstructed later to embed in a future announcement.

The solution implemented here: a second, independent per-grant credential
that the application can *recompute* rather than store. `deriveWithdrawToken`
(`backend/internal/app/consent.go`) is HMAC-SHA256 keyed on the server's
session secret, over the grant id — deterministic, not a random draw.
`mintWithdrawToken` derives it and returns both the raw token and its
SHA-256 hash; only the hash is ever persisted, in the new
`consent_grants.withdraw_token_hash` column, using the same
hash-and-look-up pattern `verification_token_hash` already uses.
`handleConfirmConsentGrant` mints and stores this hash at verification
time. Dispatch (see below) recomputes the same raw token on demand to
build the link, and also mints the hash on the fly as a backstop for any
verified grant that predates this column or otherwise lacks one. The
public `POST /api/public/consent/{token}/withdraw` route now matches
either `verification_token_hash` or `withdraw_token_hash`, so both the
original one-time link (if the recipient kept it) and every announcement's
withdraw link resolve to the same withdrawal.

An attacker who reads the database still cannot derive a usable token:
doing so requires the session secret, not just the grant id.

## API

All new routes are under
`/api/workspaces/{workspaceID}/announcements`, behind the new
`manage_announcements` permission (owner and organizer — see
[`authority-model.md`](authority-model.md)).

- **`POST .../announcements`** — draft. Body `{subject, body}`, both
  required non-empty. Never resolves an audience or enqueues anything.
- **`GET .../announcements/{id}/preview`** — returns `{subject, body,
  recipientCount, estimatedCostCents}`. `recipientCount` is the current
  count of verified, unwithdrawn `announcement`-purpose grants for the
  workspace whose address is not currently suppressed; `estimatedCostCents
  = recipientCount * ANNOUNCEMENT_UNIT_COST_CENTS`. Never returns an
  address. This count can change between preview and dispatch — that is
  expected; dispatch re-derives it fresh (see below), it never trusts a
  preview snapshot.
- **`POST .../announcements/{id}/schedule`** — body `{scheduledFor}`
  (RFC3339). Requires a future timestamp; a past or present timestamp is
  `400`. Sets `status = scheduled`. Allowed from `draft` or an
  already-`scheduled` row (reschedule).
- **`POST .../announcements/{id}/cancel`** — allowed only while `draft`
  or `scheduled`; `409` once dispatch has started (`dispatching` or
  `dispatched`) or if already `cancelled`. Records `cancelled_at` /
  `cancelled_by_person_id`.
- **`GET .../announcements`** / **`GET .../announcements/{id}`** — list
  and get, each including `deliveryStatusCounts`: outcome counts (e.g.
  `{"accepted": 12, "withheld_consent": 1}`) joined from
  `email_outbox.delivery_status` via `announcement_deliveries` — counts
  only, never a recipient address.

## Dispatch

A `-announce` mode on `backend/cmd/email-deliver` (`RunAnnouncementDispatch`,
`backend/internal/app/announcement_dispatch.go`) claims due, `scheduled`
announcements with `for update skip locked`, one at a time, each fully
inside its own transaction:

1. Claim: `scheduled` rows with `scheduled_for <= now()`, oldest first;
   mark `status = dispatching`.
2. Re-derive the raw candidate audience fresh, inside the same
   transaction — every verified `announcement`-purpose grant for this
   workspace, regardless of its current withdrawn or suppressed state.
   This is deliberately not filtered to "currently eligible" in SQL: the
   next step is where withdrawal/suppression are actually decided, so
   that a grant withdrawn between scheduling and dispatch is counted as
   withheld rather than silently vanishing from every count.
3. For each candidate recipient, call `checkSendPermission` (the same
   function `consent.md` defines and `processEmailDeliveries` already
   calls) as the final consent check. A denial (no verified/unwithdrawn
   grant, or suppressed) counts toward `withheld_count` and enqueues
   nothing for that recipient.
4. For each allowed recipient: build the withdraw link (see above),
   append it to the announcement body, enqueue one `email_outbox` row
   with `purpose = 'announcement'` and `workspace_id` set (via a
   dedicated insert, `enqueueAnnouncementEmail` — it does not reuse
   `enqueueEmail`, whose insert defaults `purpose` to `'transactional'`
   and leaves `workspace_id` null), and record one
   `announcement_deliveries` row linking it to the announcement.
5. Set `recipient_count` (enqueued), `withheld_count`,
   `estimated_cost_cents` (`recipient_count * ANNOUNCEMENT_UNIT_COST_CENTS`),
   `status = dispatched`, `dispatched_at = now()`, all in the same
   transaction, then commit.

Actual provider sending stays entirely with the existing
`processEmailDeliveries` worker (`-send`): dispatch only enqueues rows.
Enqueued rows start `held` when `MAIL_DELIVERY_ENABLED` is false (the
default) and `pending` when true, exactly like every other `email_outbox`
insert. `processEmailDeliveries` calls `checkSendPermission` again,
immediately before sending each claimed row (the existing send-time
recheck from `consent.md`), so a grant withdrawn between dispatch and
actual send is still withheld at send time, not just at dispatch time.
Dispatch never contacts the mail provider and never sends live mail in
tests: `MAIL_DELIVERY_ENABLED` stays `false` by default, and the
synthetic journey test below uses a fake sender.

Run it with `email-deliver -announce [-watch] [-limit 1..100]`; `-send`
and `-announce` are mutually exclusive on one invocation (dispatch and
provider sending are separate concerns run as separate batches/workers).
The optional `announcement-workers` Compose profile runs
`email-deliver -announce -watch`, alongside (not instead of) the existing
`mail-workers` profile's `email-deliver -send -watch`.

## Synthetic journey (proof)

`backend/internal/app/announcement_integration_test.go`, run against
disposable PostgreSQL:

- `TestAnnouncementSyntheticJourney` — grants two recipients, confirms
  both, drafts and schedules an announcement into the past-due window,
  confirms preview shows `recipientCount = 2`, withdraws one recipient's
  grant, runs dispatch (`RunAnnouncementDispatch`) and confirms exactly
  one `email_outbox` row with `purpose = 'announcement'` and
  `withheld_count = 1`, then runs the delivery worker
  (`processEmailDeliveries`) with a fake sender and confirms the one
  allowed row is accepted while the withdrawn address never receives
  anything, and confirms the get endpoint's `deliveryStatusCounts`
  reflects the one accepted outcome.
- `TestAnnouncementCancelBeforeDispatchProducesNoRows` — cancelling a
  due, scheduled announcement before a dispatch run claims it produces
  zero `email_outbox` rows and zero dispatch claims; a second cancel is
  `409`.
- `TestAnnouncementRequiresManageAnnouncementsPermission` — a plain
  member (`crew`, no `manage_announcements`) gets `403` from every
  announcement endpoint.
- `TestAnnouncementScheduleRejectsPastAndPresent` — a past `scheduledFor`
  is `400` from the endpoint itself.
- `TestAnnouncementWithdrawLinkTokenWorksThroughPublicRoute` — the
  deterministic withdraw token derived by `deriveWithdrawToken` resolves
  through the existing public withdraw route.

## Pilot gate

A limited permissioned pilot (sending to a small, explicitly approved
recipient list) requires **#7 (live deliverability)** to be verified
first — this slice's tests use a fake sender and no Resend account or
credential, matching `transactional-email.md`'s existing boundary. Do not
run a live announcement send before #7 is complete and an approved
recipient list exists.

## Remaining limits

- No SMS or other second channel; `consent_grants.channel` remains
  constrained to `'email'` (unchanged by this slice).
- `ANNOUNCEMENT_UNIT_COST_CENTS` defaults to `0` ("unknown"), not a real
  observed cost; nothing in this slice supplies a non-zero default.
- No UI: every endpoint above is API-only.
- No re-send, editing-after-schedule, or per-recipient personalization
  beyond the withdraw link; the body an operator drafts is sent verbatim
  (plus the withdraw link) to every allowed recipient.
- No rate limiting or send-window/throttling beyond what
  `processEmailDeliveries`'s existing batch limit and lease already
  provide; a workspace with a very large verified audience dispatches (and
  enqueues) its entire audience in one transaction per announcement.
- No live deliverability test and no permissioned pilot has been run; see
  "Pilot gate" above.
