# Subcult OS platform consolidation plan

Date: 2026-09-20
Status: architecture accepted; BASE-01, API-01, DB-01 and INV-01 complete at their documented evidence levels; consolidation not implemented.

## Goal

Make Subcult OS the clean Subcult.tv platform core with one Go API, one user/session authority and one PostgreSQL database. Selectively extract only current, independently testable AT Protocol, identity, privacy, cultural-model and conformance capabilities from the older Subcults repository.

## Canonical specification

Read the [development handoff](../../development/README.md), [architecture](../../development/architecture.md), [selective extraction inventory](../../development/extraction-inventory.md), [decision register](../../development/decisions.md) and [backlog](../../development/backlog.md).

## Ordered implementation units

1. BASE-01: preserve exact repository and verification evidence. Complete.
2. API-01: separate the public event projection from private operator DTOs. Implemented and focused-test-verified.
3. DB-01: establish a clean ordered, replay-safe, failure-safe OS schema and migration system. Complete with focused PostgreSQL and backup/restore evidence.
4. INV-01: finish the file-level Subcults extraction audit; no code import during inventory. Complete; no source was imported.
5. IDENT-01 and AT-01: build the canonical identity/session foundation and minimal Go AT kernel independently.
6. MODEL-01: add only the cultural records and operator/public event relationship required by accepted journeys.
7. DISC-01: ingest and project accepted public records with cursor, provenance, deletion and quarantine.
8. PUB-01: add explicit authorized publication and exact-CID reconciliation.
9. UX-01: complete unified account, organizer and attendee browser/mobile journeys.
10. QUAL-01: qualify a protected consolidated pilot before any external-state mutation or destructive cleanup.
11. COMMONS-01: extract independently reusable conformance fixtures after demonstrated use.
12. LIFE-01 and OFFLINE-01: later discovery, not hidden first-release commitments.

CONSENT-01 defines the no-inference delivery boundary alongside these slices; it does not authorize sending.

## Extraction rule

Do not merge repository histories or import Subcults wholesale. Each admitted capability must have a product journey, exact source revision/path, disposition, dependency/provenance review, minimal contract, focused OS tests, rollback, and a list of intentionally excluded behavior. Prefer rewriting an invariant behind an OS-native interface when old code is coupled to retired tables, routes, features or services.

## Prototype replacement and rollback

Do not preserve prototype routes, DTOs, tables, identifiers, passwords, sessions or client flows by default. Change API, web and mobile together against the clean model. Before discarding any database, inventory it read-only and create migration work only for specifically retained real data or external state. A code rollback cannot undo a PDS write; publication requires reviewed compensating behavior and reconciliation. Subcults remains untouched until replacement qualification and separately authorized retirement.

## Verification

Per-item checks and gates are in the [backlog](../../development/backlog.md) and [verification matrix](../../development/verification.md). Go/TypeScript AT conformance fixtures supplement, rather than replace, browser, database, provider and authoritative-PDS evidence.

## Authority boundary

This plan does not authorize deployment, provider calls, real-user migration, PDS writes, Subcults cleanup, hosted issue creation or repository archival.
