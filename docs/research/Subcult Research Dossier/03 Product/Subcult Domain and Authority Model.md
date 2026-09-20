---
type: architecture-proposal
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Domain and Authority Model

## Separate concepts before sharing interfaces

| Object | Meaning | Proposed authority |
| --- | --- | --- |
| Person | Internal account/person reference | Authentication service and verified linking |
| DID | AT account identity | DID/account control, not email equality |
| Profile / Act | Public identity / creative project | Explicit authorized publisher |
| Scene | Cultural community/context | Scene governance; separate private membership |
| Workspace | Private team and billing boundary | Workspace authorization service |
| Event | One real occurrence | Named public editor/controller |
| Appearance | Act participating in an event | Agreed attribution/correction policy |
| Tour | Group of appearances | Tour editor; does not own local finances |
| Signal | Versioned invitation | Sender with separate delivery authorization |
| Order / ticket | Purchase and entitlement | Ticket provider or qualified commerce service |
| Operations record | Staffing, commitments, closeout | One owning workspace |
| Archive | Private continuity plus optional public material | Separate review and publication authority |

This refines existing domain material described in [[Subcult Repository Evidence]]. It is a proposal for integration, not a claim that both codebases currently enforce the same contract.

## Identity and deduplication

Link accounts only after authenticated proof for both identities. An identical email string is insufficient. Preserve internal IDs when a handle changes and record the current DID linkage with an audit trail.

Do not merge events because the title/date/location look similar. Two nights of a festival can legitimately have similar descriptions. Maintain source identifiers and an explicit reviewed equivalence mapping. Conflicts should remain visible rather than silently resolved by the latest import timestamp.

A public artist, scene and venue can all reference the same occurrence. Public host relationships are credits/context, not authorization grants.

## Independent state machines

| Concern | Illustrative states | What a transition must not imply |
| --- | --- | --- |
| Public occurrence | announced, postponed, cancelled, completed | Does not automatically refund orders |
| Public publication | draft, pending, published, conflict, failed | Pending projection does not mean the event never existed |
| Private operations | planning, staffed, running, closing, closed | Closed does not authorize public financial disclosure |
| Order | pending, paid, refunded, disputed | Paid does not grant marketing consent |
| Admission | eligible, checked-in, revoked | A public RSVP is not a valid paid credential |

For a reschedule, retain the occurrence identity when policy says it is the same show, preserve old and new times for audit, update ticket-provider obligations separately and send only appropriate notices. For a replacement show, create a new occurrence and explicit relationship instead of rewriting history.

## Multi-team collaboration

Start with one operations owner and explicit event-scoped collaborators. A scene is not automatically a legal merchant. A touring artist's team may confirm an appearance without seeing the venue's supplier invoices.

More complex shared ownership is a future capability requiring conflict resolution, recovery and financial responsibility rules. Do not hide that complexity in a shared login.

Related: [[Subcult Integration Architecture]], [[Subcult Organization and OAuth Research]], [[Subcult Operations and Commerce]].

