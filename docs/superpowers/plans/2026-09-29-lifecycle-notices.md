# LIFE-01 listing-change notices

Implement owner review and approval for an already saved occurrence cancellation
or reschedule. This does not cancel the operator event, change admission, transfer
tickets, publish records or refund money.

Implemented locally: an owner-only preview endpoint, atomic notice approval and
recipient/outbox ledger, send-time guards, and private worklist outcome controls.
It checks exact saved listing status/revision/CID, renders a server template,
selects up to 500 deduplicated operational recipients and reports suppression.
The digest includes content, sorted audiences, recipient relationships and
suppression. Previewing does not change schema, enqueue mail or dispatch actions.

Implemented workflow:

1. Add an immutable notice approval and recipient ledger. A server-generated
   preview binds exact occurrence revision/CID, templated public listing content,
   selected operational audiences, normalized recipients and source relationships.
   Approval compares the digest under lifecycle locks, creates the queue action
   and recipient/outbox rows atomically, and supports exact request-key replay.
   One notice per occurrence revision prevents a new identity from blindly
   resending an uncertain earlier notice.
2. Recheck current decision, owners, occurrence and source relationship immediately
   before sending. Preserve suppression and existing provider idempotency/retry
   windows. Denial before the first provider attempt is withheld; denial after a
   prior attempt is quarantined for reconciliation. Queue success and per-recipient
   provider acceptance/feedback remain separate facts.
3. Add private owner preview/approval/results controls to the worklist. Clear
   private state on authorization loss or workspace/event changes. Require
   re-preview after a stale digest; preserve request identity across lost replies.

Verification: disposable PostgreSQL regressions for stale revision/CID/audience,
deduplication, approval replay/conflicts, suppression, authority/relationship loss,
provider failures and migration preservation; frontend rendering/contract checks;
synthetic browser review/approval with sending disabled; full repository verify
and DB gates, plus focused lifecycle/mail-worker race tests.

Migration 26 adds notice/recipient ledgers and the authority withholding status;
prior held messages and drafts are preserved. Older binaries reject schema 26.
A deployment candidate needs a qualified pre-migration backup/recovery path.

Remaining: real provider delivery to approved test recipients, operator
reconciliation controls for uncertain outcomes, browser qualification of the
approval journey, and the wider coordinated operator-event lifecycle.
No live messages or deployment are part of automated qualification.
