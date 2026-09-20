---
type: contribution-proposal
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Sync and Projection Contributions

## Current infrastructure

Tap already addresses authenticated synchronization, backfill and recovery, whereas Jetstream is designed for simpler event consumption with different trust assumptions. Its official introduction describes verified filtered events, repository recovery and acknowledged delivery options. Do not propose “invent reliable AT sync” as new work. [Introducing Tap](https://atproto.com/blog/introducing-tap).

Subcults already treats Tap and Jetstream as distinct components in its qualification plan. The most useful contribution is likely an application-facing conformance harness or a concrete upstream bug reproduction.

## Proposed harness

Use synthetic repositories and a disposable application projection. Record the intended authoritative state, delivered event sequence, acknowledgements, crashes and final query results. The assertion is convergence to the correct public state, not simply receipt of a message.

Separate sync correctness from side-effect correctness. Rebuilding a projection must not resend old campaigns, recreate payments or duplicate private operations records. A replayed public event is not authorization for another external action.

## Cases

- Start tracking a repository with existing records and concurrent live edits.
- Receive duplicate deliveries and restart after applying but before acknowledging.
- Recover after disconnect beyond the usable replay window.
- Process deletion, account deactivation/reactivation and migration.
- Reject stale content revisions without resurrecting deleted occurrences.
- Handle malformed/unsupported schema records without stopping unrelated ingestion.
- Rebuild into an empty projection and compare canonical event identities/status.
- Restore backups and reconcile blob/media references separately.
- Measure lag and expose stale-state warnings to users.

Expected semantics must be derived from the deployed Tap/protocol versions. Do not invent universal ordering guarantees beyond those documented and tested.

## Contribution package

Provide a minimal reproduction, pinned versions, synthetic fixtures, expected/actual outcome and tests. Keep a reusable fixture library separate from application-specific SQL. Offer documentation for choosing verification and delivery modes.

If the issue is our projection handler, fix it locally rather than filing it as a protocol defect. If the shared tool behaves correctly but documentation is confusing, contribute the explanation and example.

## Business value

Correct cancellations and account continuity are practical user benefits. A disappearing or resurrected show undermines trust. The harness also reduces operational support cost by making recovery repeatable.

Related: [[Subcult Integration Architecture]], [[Subcult Release Qualification]], [[Subcult Repository Evidence]].

