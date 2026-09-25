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
Only the occurrence-edit behavior in the next section ships in this slice.

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

## Ledgered external action intent (dispatch not implemented)

Migration 000016 now provides the private decision and action ledger only. It
can persist an approved cancellation/reschedule decision and an action intent,
claim one due intent with a fenced lease, retain retry scheduling, and
supersede unsent intents. It does not expose an HTTP cancellation route, change
an event or occurrence, create recipients, send a notice, write a public
record, call a provider, or issue a refund. An expired running lease becomes
`unknown`, never a blind retry; a later orchestration slice must reconcile that
destination before it can create another external action.

The ledger verifies that a change's event and approving active owner belong to
the recorded workspace. It stores a bounded object snapshot and bounded action
payload, but it does not enforce snapshot immutability at the database layer.
The future API and approval flow must append a new approved decision rather
than alter a recorded decision or action identity. Failure categories are a
small internal code set; provider error text and unbounded response bodies do
not belong in this ledger.

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
| Action-intent fencing, retry scheduling, lease expiry, supersession and upgrade | `TestLifecycleActionLedgerIsIdempotentFencedAndNeverBlindRetriesUnknownWork`; `TestLifecycleActionConcurrentClaimHasOneWinnerAndSupersedeStopsPending`; `TestLifecycleActionCreateThenSupersedeSerializesOnChange`; `TestLifecycleActionMigrationUpgradesVersionFifteen` | Source coverage added; the corrected upgrade fixture awaits the next combined disposable-database gate |
| External notice failure/retry and reconciliation | Future coordinated-action dispatch tests | Not implemented; existing mail-worker tests do not prove this workflow |
| Payment/refund ambiguity | Future #53 provider test-mode journey | Not qualified |

Focused command, inside the installed disposable database test environment:

```sh
cd backend
go test -race ./internal/app -run 'TestOccurrenceLifecycle|TestEventOccurrence' -count=1 -v
```

Run `make verify` and the full disposable `make test-db` gate before delivery.
No migration is required. Rolling back this slice removes edit safeguards;
it does not undo accepted edits. #50 remains open for the coordinated-action
and notice-failure proof; no user-facing cancellation/refund workflow is claimed.
