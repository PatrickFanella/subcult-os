---
type: architecture-proposal
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Integration Architecture

## Recommended boundary

Keep Subcults as the starting authority for public identities, scenes, occurrences and AT publication. Keep Subcult OS as the private operations service. Present them through a coherent product, but avoid merging repositories before their contracts are stable.

This is a recommendation based on source fit, not a declaration that either system is ready for production. [[Subcult Repository Evidence]] distinguishes implementation assets from qualification.

## First integration contract

Maintain a mapping from public occurrence ID and optional AT URI to exactly one private operations record for the owning workspace. Store source revision, mapping creation actor, update history and conflict state.

Use authenticated server-to-server calls with narrow capabilities. Enforce workspace/event access on every private action; possession of a public URI is never authorization. Keep provider credentials and OAuth sessions out of browser-visible records.

Define which service owns each writable field. A public title correction may update discovery; it must not replace private notes. A settlement adjustment must not generate an AT publication.

## Consistency and failure handling

Write financial/operational changes transactionally in their authoritative service. For public updates, use an outbox and a durable publication status, then reconcile against the accepted PDS record and projection. Preserve idempotency keys and revision conflicts.

An accepted remote write followed by a timeout is an **unknown outcome**, not necessarily a failure. Reconcile before retrying. An indexed projection lag should show “publishing” or “syncing,” not invite the user to create duplicates.

Importing provider orders requires their stable IDs and webhook/reconciliation rules. If no API access is authorized, begin with a link and dated manual aggregate import rather than simulated synchronization.

## Migration sequence

1. Document identity linking, event mapping and field authority.
2. Introduce read-only cross-links and test access denial.
3. Enable one controlled event-to-operations creation flow.
4. Reconcile event corrections, cancellation and deletion without private leaks.
5. Add the narrow announcement journey.
6. Prove recovery and export before broad customer migration.

Keep rollback possible: preserve original records, retain mapping history, and avoid destructive backfills. No automatic repository consolidation or namespace migration is authorized by this dossier.

## Acceptance cases

A public co-host cannot read settlement. A user removed from a workspace loses private access even while still publicly credited. A PDS outage does not invalidate a successful order. A retry does not create two operations records. A cancelled provider event is not resurrected by a stale import. An account move does not attach records to a different person.

Related: [[Subcult Domain and Authority Model]], [[Subcult Sync and Projection Contributions]], [[Subcult Release Qualification]].

