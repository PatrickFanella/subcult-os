# ADR 0006: Do Not Preserve Prototype Compatibility by Default

## Status

Accepted

## Context

Subcult OS and Subcults were not operated as production systems with a supported external API, committed database compatibility policy, or established population whose identifiers and client behavior must be migrated in place. Treating their current REST routes, DTOs, tables, identifiers, authentication flows, or frontend assumptions as permanent contracts would carry prototype decisions into the new platform without a user or operational need.

## Decision

The consolidated Subcult.tv platform may make deliberate breaking changes to prototype APIs, schemas, identifiers, authentication, clients, and internal workflows. New work targets the clean architecture rather than building dual-read, dual-write, route-alias, or legacy-session compatibility by default.

Before discarding any local or hosted data, perform a read-only inventory and identify whether it is synthetic, reproducible, or irreplaceable. Preserve and migrate only specifically identified real user-owned data or external state. Destructive cleanup still requires explicit authorization.

Compatibility remains mandatory where an actual external authority exists, including published AT records that must remain addressable, completed or pending provider transactions, legal/audit obligations, issued credentials still in use, and any data the product owner explicitly designates for retention.

Privacy, authorization, consent, protocol correctness, provenance, idempotency, and failure-recovery invariants are requirements, not legacy compatibility.

## Consequences

- DB-01 may establish a clean versioned schema rather than adopting the embedded alpha schema as migration zero.
- Identity may be replaced cleanly instead of supporting a legacy password/session transition.
- Web and mobile clients may change with the API in the same implementation slice.
- Existing tests are retained only when they prove a current invariant; snapshots of obsolete behavior may be removed deliberately after replacement coverage exists.
- Migration code is written only for inventoried data that actually needs migration.
- Rollback protects external writes and newly created persistent state; it does not promise restoration of every prototype interface.
