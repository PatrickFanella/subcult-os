---
type: qualification-plan
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Release Qualification

## Distinguish release evidence from research

This dossier does not qualify a deployment. The existing source and historical QA documents provide a starting inventory, not current proof that an event can safely depend on the combined application.

The root project asks for `make verify`. Packaging re-attempted it with a bounded timeout; see [[Subcult Research Methods and Refresh]] for the exact artifact-work result. Never present documentation link validation as an application test.

## Required evidence layers

| Layer | Evidence required |
| --- | --- |
| Static/contracts | Current source revision, schema/contracts and private/public field rules |
| Local automated | Relevant backend/frontend/mobile tests, lint/build, failure cases |
| Database/integration | Real configured transaction, migration and rollback tests |
| Provider | Approved payment/messaging/PDS tests with exact configuration and outcomes |
| Browser/device | Participant, organizer and crew journeys at supported viewport/device |
| Recovery | Backup restore, reconnect, idempotency and access revocation |
| Event operation | Rehearsed fallback and observed supported live workflow |
| Repetition | Multiple event cycles, support load and incident review |

Each result must name revision, environment, date, expected behavior, actual outcome and limitations.

## Critical test stories

A removed crew member cannot read a private roster. An opted-out contact is suppressed after a message was scheduled. Public publication cannot leak a protected address. A successful provider payment survives a delayed AT projection. A cancelled occurrence is not restored by replay. A restore brings back the correct operational state without sending duplicate announcements.

For native door operation, test offline/online transitions, duplicate scans, revoked tickets and device loss. If not qualified, use the existing ticket provider's workflow and state that boundary.

## Existing Subcults gates to refresh

The historical release material names provider/device/browser evidence, strict invite expiry, dedicated capacity/restore and a seven-day Tap/Jetstream parity requirement. Do not assume a short local run satisfies those gates or erase them because the proposal expanded.

Invite expiry needs deployed-version evidence; upstream disablement exists, so characterize the precise gap. See [[Subcult PDS Invite Research]].

## Stop and rollback

Define who can stop messaging, disable publication or switch the door fallback. Keep revisioned rollback artifacts and restore instructions. Preserve failed evidence rather than rewriting a report after a repair.

A passing rerun is evidence for that rerun; it does not retroactively turn a failed event or interrupted soak into a success.

Related: [[Subcult Risk Register]], [[Subcult Repository Evidence]], [[Subcult Metrics and Measurement]].

