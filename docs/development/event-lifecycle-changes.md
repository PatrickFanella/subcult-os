# Cancellation and rescheduling (LIFE-01)

Issue #50 was promoted for development by the owner on 2026-09-24 as part of
the #50–71 expansion. This document specifies the coordinated workflow and
identifies what the current code actually enforces. It does not enable refunds,
send notices or publish remote records.

## Current boundaries

`events` owns the private operator plan and ticket allocation. Its states are
`draft`, `published` and `end_of_night`. `event_occurrences` owns cultural listing
time/place/status, with `scheduled`, `rescheduled`, `postponed` and `cancelled`.
Several occurrences can point to one operator event. A cancellation of one
listing must not silently cancel the whole event or another host's listing.

Tickets belong to the operator event. The normal event editor refuses a start
time change on a published event after reservations exist. A public occurrence
edit does not change tickets, admission, the operator start time, commitments,
payments, notices or immutable archive snapshots. In particular, the existing
occurrence `cancelled` state is **not an operational event cancellation**.

The relevant implementations are `backend/internal/app/cultural_occurrences.go`,
`event_lifecycle.go`, `tickets.go`, `reminders.go`, `notifications.go`,
`commitments.go` and the archive handlers in `events.go`.

## State matrix

The coordinated actions below are a specification for later implementation.
Occurrence safeguards, the private worklist and notice previews are implemented
as described below.

| Requested change | Public listing | Operator plan and admission | Provider tickets and money | Crew/participant notices | Private archive |
| --- | --- | --- | --- | --- | --- |
| Correct an unpublished draft | Remains unpublished | Edit allowed fields; no implied launch | Preserve provider/payment state | No automatic notice | No mutation |
| Reschedule a listing only | Explicit new instant/zone; `rescheduled` | Unchanged; explain this is listing-only | Unchanged | None implied | No mutation |
| Cancel a listing only | `cancelled`; preserve stable identity and history | Unchanged; other listings unaffected | Unchanged | None implied | No mutation |
| Coordinated reschedule | Preview each affected listing and target time | New revision; explicitly remap affected shifts/reminders; keep tickets identifiable | Reconfirm applicability per provider; never assume free transfer or refund | Separate operational notices with per-audience outcomes | Append correction provenance; retain earlier snapshot |
| Postpone without replacement date | `postponed`; old date remains historical, not a new promise | Explicit sales/admission hold; retain allocation and assignments pending review | Preserve evidence; show provider state as pending/unknown until confirmed | Explain date unknown; do not invent a replacement date | Preserve history |
| Coordinated cancellation | Cancel selected listings with revision checks | Explicit cancellation state blocks new sales and admission; retain prior check-ins | Cancellation and refund are separate actions; pending payments require reconciliation | Queue notices separately; failure must remain visible | Preserve closeout/obligations; cancellation is not settlement |
| Reinstate a cancelled/postponed event | New explicit approval and revision | Recheck capacity, authority, shifts and admission | Reconfirm validity; never undo a completed refund implicitly | New notice; do not retract an already delivered notice | Append correction, never erase history |

For multi-occurrence events, approval must select whether the change affects one
listing or the underlying operator event. No default may silently choose all
linked listings. Existing financial commitments remain obligations until an
authorized correction records a reason; cancelling an event does not mark them
paid, forgiven or refunded.

## Occurrence API safeguards implemented here

`PATCH /api/events/{eventID}/occurrences/{occurrenceID}` accepts two optional
preconditions alongside its existing fields:

- `expectedUpdatedAt`: the exact `updatedAt` returned by the preview/read. A
  malformed timestamp returns 400; an outdated revision returns 409.
- `expectedPublicCid`: the observed public CID. An empty string means no public
  CID was recorded. A mismatch returns 409. This is a comparison against the
  locally recorded CID, not a fresh remote observation or an AT repository CAS.

The final SQL update also compares the loaded revision and CID, so an edit
racing between read and write cannot overwrite the winner. Clients without
preconditions retain their existing API shape, but cannot detect an edit made
before their request began. New preview/approval flows must provide both values
and reload/re-preview after 409. Publication will additionally require the
remote repository's conditional-write check under #19/#20.

The update advances `updatedAt` by at least one microsecond, even if the database
clock repeats or moves backward. It therefore serves as a monotonic edit cursor
for this endpoint, not an exact wall-clock audit timestamp during clock skew.
Audit records retain their separate timestamps.

Creation and editing reject unknown timezones and the process-dependent `Local`
zone. An omitted or cleared zone remains unknown; no venue zone is invented.
Known zones use the Go IANA database, embedded for the minimal runtime image.
Any specified end must be strictly after the start. Invalid retained schedules
must be corrected, or their optional end/zone explicitly cleared, before another
edit succeeds. No historical rows are rewritten by this change.

Times remain RFC3339 absolute instants with explicit offsets, rendered in the
selected zone. They are not parsed as naive wall times. A future wall-time
editor must reject spring-forward gaps and require an offset choice for repeated
fall-back times. It must not silently normalize either case. All-day date-only
calendar export semantics remain part of #22/#59.

Changing start/end automatically marks a scheduled occurrence as rescheduled;
clearing an existing end also counts as a schedule change. Changing dates on a
cancelled/postponed occurrence does not reactivate it. Explicit status changes
remain available through the existing API.

## Ledgered external action intent

Migration 000016 now provides the private decision and action ledger only. It
can persist an approved cancellation/reschedule decision and an action intent,
claim one due intent with a fenced lease, retain retry scheduling, and
supersede unsent intents. It does not expose an HTTP cancellation route, change
an event or occurrence, create recipients, send a notice, write a public
record, call a provider, or issue a refund. An expired running lease becomes
`unknown`, never a blind retry; a destination-specific workflow must reconcile that
destination before it can create another external action.

The ledger verifies that a change's event and approving active owner belong to
the recorded workspace. It stores a bounded object snapshot and bounded action
payload, but it does not enforce snapshot immutability at the database layer.
The future API and approval flow must append a new approved decision rather
than alter a recorded decision or action identity. Failure categories are a
small internal code set; provider error text and unbounded response bodies do
not belong in this ledger.

## Destination-scoped dispatch infrastructure

Migration 000025 adds `dispatch_approved`, defaulting to false for every existing
and newly inserted HTTP worklist draft. The internal action-creation helper can
create an explicitly approved action; replaying a draft's idempotency key with
that flag changed is a conflict. No HTTP approval route, runtime adapter or
worker is installed by this slice.

`dispatchLifecycleAction` consumes at most one approved action for an adapter's
exact action kind and destination. Other destinations and all drafts remain
untouched. The adapter validates its complete payload locally before the
dispatcher rechecks the persisted lease, decision status, approving owner's
current membership and exact occurrence revision/recorded CID. Missing legacy
snapshot bindings fail validation. Failed preconditions record a terminal
`permission` or `validation` outcome without invoking the adapter.

Adapters receive the stable action idempotency key and a context bounded to
45 seconds, below the two-minute lease. Each adapter must supply an attempt
budget from one to ten; an explicitly retryable result at that limit becomes
terminal `failed`, retains its sanitized failure category and clears the retry
schedule. Reducing the budget below an already claimed attempt rejects execution.
Adapters return a sanitized outcome and
optional bounded provider reference. Explicit retryable results require a future
retry time and no provider acceptance reference. Adapter errors become `unknown`
with the fixed `transport` category; malformed results become `unknown/internal`.
Neither is reclaimed automatically. A bounded adapter-supplied provider reference
is retained internally for reconciliation, including uncertain acceptance; raw
adapter errors are never persisted. A cancelled caller still permits a bounded
attempt to persist the outcome, and expired leases cannot complete an action.
The private worklist API exposes dispatch approval, retry time, completion time
alongside each action's independent status and attempts. Provider references
remain internal, as required by the existing API contract.

These are local checks immediately before invocation. They do not lock a remote
destination or prevent authority/revision changes after the check. A future
adapter must honor cancellation, use provider idempotency/conditional writes,
recheck its destination authority and reconcile ambiguous acceptance. Before an
operational notice adapter is installed, its separate approval flow must snapshot
reviewed content and relationship-derived recipients, enforce suppression, and
track per-recipient delivery rather than treating enqueue as delivery success.

## Private operator worklist

Owner-only `GET` and `POST /api/events/{eventID}/lifecycle-intents` expose the
ledger as a private, `Cache-Control: private, no-store` worklist. A decision
must name an existing occurrence in that event and submit its exact
`updatedAt` and recorded public CID. The server locks the event, validates an
active owner again inside the transaction, locks the occurrence, and rejects a
stale revision or CID with 409. It derives the decision snapshot and bounded
action payload itself; callers choose only from the four ledger action kinds.
URLs, provider credentials, recipients, notice content, refund amounts, and
provider responses are not accepted by this route.

The request carries a UUID `decisionKey`. Retrying the same key returns the
same decision only when its workspace, event, occurrence revision, decision,
reason, owner, and action kinds are identical; a changed binding is 409. A
newer decision can supersede only `pending` and `retryable` drafts. `running`,
`unknown`, succeeded, and failed actions retain their existing state; unknown
work still requires destination reconciliation. The worklist UI labels every
decision as recorded and keeps provider/publication/refund drafts separate from
explicit notice queue approval.

A coordinated change needs one private change record with actor, workspace,
reason, scope, old/new values, revision and approval digest. Persist the local
decision and its action intents atomically. Each affected destination then has
its own action record: public record write, provider ticket change, operational
notice, and any separately approved refund. A failure in one action cannot
cause the others to be silently replayed.

Use stable action identity and states `pending`, `running`, `succeeded`,
`retryable`, `unknown`, `failed` and `superseded`. Record bounded attempts,
next-attempt time, sanitized failure category and provider reference. `unknown`
means a timeout may have occurred after success; reconcile by provider identity
before retrying. Fence stale worker claims and recheck current permission and
change revision at dispatch. A newer approved change supersedes unsent older
notices; delivery already accepted by a provider requires a correction notice.

Notice delivery never controls whether cancellation was recorded. The operator
must see cancellation recorded with notice delivery pending/failed. Ticket and
crew recipients must derive from their operational relationship, not from an
announcement audience or an inferred marketing grant. Keep announcement consent
and suppression enforcement intact; operational notice policy and recipient
selection need their own reviewed implementation.

## Listing-change notice preview

Owner-only `POST /api/events/{eventID}/lifecycle-intents/{changeID}/notice-preview`
accepts `audiences` containing `ticket_holders`, `assigned_crew`, or both. It
requires an approved decision with an operational-notice action, a still-active
original owner, and the exact saved listing revision and recorded CID. The
listing must already be cancelled or rescheduled to match the decision. A
superseded decision, stale revision/CID or missing CID condition returns 409.
The response is private and must not be cached.

The server renders the listing name and, for rescheduling, its start/end/zone.
The message explains that it concerns the listing only; tickets, payments,
refunds, crew assignments and the operator plan remain separate. Private
reasons, application messages and staffing notes do not enter its template.

Ticket recipients have a reserved/checked-in ticket with free, pending or paid
payment status. Crew recipients have an assigned staffing item and either active
workspace membership or an accepted/confirmed application for this event.
Addresses are normalized and deduplicated. Suppressed addresses remain visible
as withheld candidates. Invalid addresses or more than 500 distinct recipients
block review. Contacts and announcement-consent audiences are not consulted.

A SHA-256 digest binds the content, revision/CID, sorted audience selection,
normalized recipient addresses, source identities and suppression state from
one repeatable-read snapshot. It is a review fingerprint, not a sending grant.
Future approval must recompute it under its own locks. The worklist clears its
preview when audiences/event change and clears private state on access denial.

The preview endpoint writes no ledger or outbox rows and dispatches no actions.

## Listing notice approval and outcomes

Owner-only `POST /api/events/{eventID}/lifecycle-intents/{changeID}/notice`
accepts a UUID `requestKey`, the exact `previewHash` and the reviewed audiences.
It accepts no caller-supplied recipients or message. The transaction locks the
event, decision, active owners and occurrence, rebuilds the preview, and rejects
changed content, relationships or suppression with 409. Approval atomically
writes the notice, per-recipient ledger, outbox rows and a succeeded local queue
action. The queue action's success means enqueueing completed; provider delivery
has separate state. Superseding a decision leaves that receipt intact and blocks
further sending through the worker's authority recheck.

Exact request-key replay returns the same notice before checking later listing
changes. A changed request binding conflicts. One notice per decision and per
workspace/occurrence revision prevents a new identity from blindly resending an
uncertain earlier message. Original decision-key replay still compares only its
original draft action set. Zero recipients block approval; a suppressed recipient
gets a withheld ledger row without an outbox row.

Eligible messages use transactional purpose and the existing mail worker.
Sending-disabled approvals create held rows. Enabling mail delivery does not
release existing held rows. Enabled sending retains the configured Resend,
sender and webhook-secret requirements; automated qualification uses fake
provider functions and no live credentials. Announcement consent is unchanged.

Immediately before each provider call, the worker rechecks both owners, decision
status, exact occurrence revision/CID/status, approved message/outbox binding,
and the chosen ticket or crew relationship/email. Existing suppression applies
independently. First-attempt authority denial clears the body and records
`withheld_authority`. Denial after an earlier attempt records `quarantined`;
acceptance may already be uncertain, so it never asserts that the notice stayed
unsent. Database errors retain the lease/body for recovery. Retries preserve the
same outbox identity and the existing eight-attempt/23-hour bounds.

Owner-only `GET` on the notice route returns its approved content and recipient
statuses, attempt counts and feedback. It excludes provider identifiers, leases
and idempotency keys. Provider `accepted` and feedback `delivered` remain separate
facts; `pending` or `quarantined` can retain unknown earlier acceptance. The UI
preserves request identity across a lost approval reply, requires a new preview
after 409, clears private state on access denial, and can inspect existing
outcomes after supersession.

Migration 000026 adds the immutable notice/recipient ledgers and the authority
withholding state. Prior held messages and draft actions remain unchanged.
Older binaries reject schema 26; rollback needs a qualified pre-migration
recovery path. No live provider delivery or operator reconciliation control is
qualified by these local tests.

## Refund gate

Automatic refunds stay unavailable until #53 supplies provider test-mode
evidence and an explicit operator-approved policy covering eligibility,
amount/currency, fees, partial refunds, timing, disputes and responsibility.
An event status edit never issues a refund. Preserve payment, cancellation and
refund as separate facts. Do not mark a refund complete from a queued request
or assume a timeout means failure.

## Acceptance evidence and remaining work

| Boundary | Executable check | Coverage |
| --- | --- | --- |
| Stale local revision and recorded CID | `TestOccurrenceLifecycleRejectsStalePreview` | 409; prior accepted state survives |
| Two edits approved against one revision | `TestOccurrenceLifecycleConcurrentPreviewHasOneWinner` | Exactly one success and one conflict |
| Clock moves backward | `TestOccurrenceLifecycleRevisionAdvancesAfterClockRollback` | Edit cursor advances strictly; old preview rejected |
| Invalid zone/interval and DST instants | `TestOccurrenceLifecycleValidatesSchedule`; `TestEventOccurrenceDSTChicagoRoundTrip` | Invalid inputs rejected; spring/fall instants preserved |
| Listing cancellation isolation | `TestOccurrenceLifecycleCancellationPreservesPrivateState` | Private event/ticket rows and email queue unchanged; date correction stays cancelled |
| Listing reschedule isolation | `TestEventOccurrenceRescheduleDoesNotChangeTickets` | Existing ticket identity/admission state retained |
| Actual remote stale CID | Future #19/#20 conditional-write test | Not implemented |
| Action-intent fencing, retry scheduling, lease expiry, supersession and upgrade | `TestLifecycleActionLedgerIsIdempotentFencedAndNeverBlindRetriesUnknownWork`; `TestLifecycleActionConcurrentClaimHasOneWinnerAndSupersedeStopsPending`; `TestLifecycleActionCreateThenSupersedeSerializesOnChange`; `TestLifecycleActionMigrationUpgradesVersionFifteen` | Durable ledger coverage; it does not prove a provider dispatch |
| Owner worklist revision/CID, idempotency, authorization and unsent supersession | `TestLifecycleIntentHTTPIsOwnerOnlyFencedAndDraftOnly`; `TestLifecycleIntentSupersedeOnlyStopsUnsentActions` | Owner-only draft visibility; stale revision/CID and changed/replayed decision keys are rejected, while `running` work stays visible for reconciliation |
| Destination selection, stable retry identity, uncertain outcomes and dispatch-time authority/revision guards | `TestLifecycleDispatchSelectsApprovedDestinationAndKeepsDraftsUnexecuted`; `TestLifecycleDispatchRetryKeepsIdentityAndUnknownNeverReplays`; `TestLifecycleDispatchStopsAfterAdapterRetryBudget`; `TestLifecycleDispatchRechecksAuthorityRevisionCIDAndPayload`; `TestLifecycleDispatchInvalidOutcomeAndExpiredLeaseRequireReconciliation` | Disposable DB with synthetic adapters; no real email, provider or publication calls |
| Draft approval boundary and migration | `TestLifecycleDispatchApprovalCannotPromoteReplayedDraft`; `TestLifecycleDispatchMigrationKeepsExistingActionsDraftOnly` | Existing drafts stay unexecuted after upgrade; changed approval cannot reuse an action identity |
| Listing notice content, audience, digest, access and revision checks | `TestLifecycleNoticePreviewRecipientsPrivacyAndDigest`; `TestLifecycleNoticePreviewAuthorityRevisionAndLimits`; `TestLifecycleNoticePreviewAssignedCrewAndSavedStatus` | Real handlers against disposable DB; preview does not enqueue or dispatch |
| Notice approval, replay, concurrent requests, rollback and upgrade | `TestLifecycleNoticeApprovalReplaySuppressionAndStaleDigest`; `TestLifecycleNoticeConcurrentApprovalHasOneBatch`; `TestLifecycleNoticeApprovalRollsBackEveryQueueRow`; `TestLifecycleNoticeMigrationPreservesLegacyMailAndDrafts` | Atomic local queue; exact replay; one batch per revision; retained legacy rows |
| Notice worker authority, ticket/crew relationship checks, stable retries and feedback | `TestLifecycleNoticeDeliveryWithholdsChangedAuthority`; `TestLifecycleNoticeDeliveryRechecksCrewRelationships`; `TestLifecycleNoticeDeliveryRetryAndFeedbackStaySeparate` | Synthetic provider calls only; no live delivery proof |
| Actual operational notice delivery/reconciliation | Future approved-recipient provider journey and reconciliation workflow | Live provider qualification and operator reconciliation controls remain open |
| Payment/refund ambiguity | Future #53 provider test-mode journey | Not qualified |

Focused commands, inside the installed disposable database test environment:

```sh
cd backend
go test -race ./internal/app -run 'TestOccurrenceLifecycle|TestEventOccurrence' -count=1 -v
go test -race ./internal/app -run 'TestLifecycle(Action|Change|Intent|Dispatch)' -count=1 -v
```

Run `make verify` and the full disposable `make test-db` gate before delivery.
Migrations 000016, 000023 and 000025 are required for the private action ledger,
stable decision-key retries and explicit dispatch approval boundary. This owner
worklist records draft, unexecuted
intents; approved listing notices have their separate local queue and mail
worker route. It does not change a public record, issue a refund or establish a
coordinated operator-event cancellation. #50 remains open for live provider
qualification, reconciliation controls and the wider coordinated workflow.
