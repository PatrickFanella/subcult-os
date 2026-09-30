# Accessibility information discovery record — 2026-09-25

Status: historical planning record for issue #55 (`ACCESS-INFO`). This record
implemented no model or outreach. The owner's later expansion-development
instruction permits a [private event worksheet](../development/event-access-information.md)
source slice. Public claims, venue inheritance, personal accommodation workflow
and accessibility-user evaluation remain unfinished. Issue #55 remains open.

## Decision boundary

The product may eventually present verified accessibility information about a
venue and a specific event. It must not turn an organizer's unreviewed note,
an old observation, or a request for accommodation into a public claim.

The prerequisite cultural model issue #12 is closed. The pilot issue #36 is
still open. Before source implementation, the owner must record all of the
following:

1. A concrete partner or event need that cannot be met safely with the current
   event description and direct contact path.
2. A rights and privacy review for the proposed sources, review workflow,
   public wording, and retention of private requests.
3. Available operational capacity to review corrections and expiry work.
4. A consented, compensated where appropriate, accessibility-user evaluation
   of the proposed display and correction path.

If that evidence does not support a useful, maintainable model, record a
no-go decision and leave issue #55 open or close it with the evidence. Do not
infer demand from this document or from a generic accessibility objective.

## Candidate assertion vocabulary

Each item below is a distinct assertion with its own source and review state.
It is not an overall "accessible" label.

| Topic | Candidate assertion | Explicit values |
| --- | --- | --- |
| Entry | Step-free route reaches the admitted entrance | `yes`, `no`, `unknown` |
| Bathrooms | Step-free bathroom availability at the venue | `yes`, `no`, `unknown` |
| Seating | Seating availability and any limits | `available`, `limited`, `not_available`, `unknown` |
| Sensory conditions | Sound, light, crowd, smoke, or scent information | structured text only when source-backed; otherwise `unknown` |
| Transit | Nearby transit and step-free route information | source-backed text or `unknown` |
| Access contact | A contact channel for access questions | present, unavailable, or `unknown` |

`unknown` is a meaningful public state. It means the system has no current,
source-backed assertion for that topic. It does not mean `no`, does not invite
guessing from photographs or prior events, and does not support a promise.

## Venue assertions and event verification

A venue assertion describes a feature attributed to the venue. An event
verification describes whether that feature is expected to apply to one event
at one time. These records must remain separate.

For example, a venue may have a step-free entrance while a particular event
uses a different door, blocks the route with load-in, or has temporary seating
constraints. A current event verification can affirm, narrow, supersede, or
leave unknown the venue assertion for that event. It must never silently
overwrite the venue history.

Each future public assertion or verification needs at least:

- its topic and explicit value;
- the asserted scope: venue or one event;
- source type and source reference or operator-recorded provenance;
- source date when known;
- reviewer and review date;
- a current-until or review-by date, if the source is time-sensitive;
- correction status and a reason when superseded or withdrawn.

When an explicit review-by or current-until deadline has passed, public display
must say that the information needs confirmation or return to `unknown`. A
review date remains historical provenance; it does not expire an assertion by
itself. A correction should retain the old record for private audit but stop
presenting it as current.

## Public and private boundary

Public event pages may show only current, source-backed venue assertions and
event verifications, with their review date and an honest unknown state.
Public display must not expose reviewer identity unless a later review approves
that disclosure, source material that contains personal data, internal notes,
or a record of someone asking for help.

Personal accommodation requests are private operational information. They need
a separate future feature with its own collection, access control, retention,
deletion, notification, and incident-review decisions. They must not be stored
inside public venue assertions, event verification notes, public event
descriptions, AT Protocol records, ticket metadata, or this proposed model.

An access contact is a published route for questions, not permission to collect
or publish a requester's disability, diagnosis, mobility information, or other
personal data.

## Source, review, correction, and expiry workflow

The future workflow should admit an assertion only after a named responsible
workspace role records its provenance and review. A useful minimal review
screen would require a reviewer to choose an explicit value or `unknown`; it
must not default an unanswered field to `yes`.

Corrections need an accessible published route, such as the access contact,
and an operator queue that can mark an assertion disputed, corrected,
superseded, or unknown. The initial implementation must not promise a response
time without staffing evidence.

Expiry needs a job or operator review list that identifies assertions whose
review-by date has elapsed. The first build should prefer conservative expiry:
remove the current claim or mark it `unknown` until it is reviewed again.
No automated refresh may infer a current condition from old source text.

## Accessibility-user validation protocol

Before implementation is accepted, conduct a small, consented evaluation with
people who use accessibility information when deciding whether to attend.
Recruitment, compensation, consent wording, and the facilitator must be
approved before contact. Do not recruit from ticket, contact, role application,
or consent-grant records merely because they contain a reachable address.

Use synthetic venues and events for the first prototype. Give participants
scenarios with `yes`, `no`, `limited`, expired, disputed, and `unknown`
information. Observe whether they can:

1. distinguish a venue assertion from an event-specific verification;
2. understand `unknown` without treating it as an assurance or refusal;
3. find source/review timing and the access contact;
4. identify an expired assertion and describe the safe next action;
5. submit or describe a correction path without disclosing unnecessary
   personal information.

Record only the minimum de-identified feedback needed for the decision. Do not
record diagnoses, accommodation requests, recordings, contact details, or
attendance intent unless separately approved. The decision record must name
the participant count, method, synthetic artifact revision, observed failures,
and resulting changes or no-go decision. Until then, no claim of meaningful
accessibility-user validation is justified.

## Proposed later implementation slices

After the decision gate, keep the work split and independently reviewable:

1. **Venue assertion ledger:** additive migration and protected operator
   workflow for source-backed assertions, review metadata, corrections, and
   expiry; no public page yet.
2. **Event verification and public serializer:** event-scoped verification,
   conservative expiry behavior, public display of current/unknown state, and
   serializer tests proving internal provenance and private notes do not leak.
3. **Private accommodation workflow:** only if a demonstrated partner need and
   a separate privacy, retention, access-control, and operational review approve
   it.

Each source slice needs disposable PostgreSQL integration tests for fresh and
upgrade paths, public serializer sentinels, correction and expiry regressions,
and `make verify`. The public display slice also needs keyboard, screen-reader
where available, and actual accessibility-user evaluation evidence. Local
tests prove implementation behavior only; they do not establish current venue
conditions or usability with the intended audience.

## Current decision

No-go for implementation at this time. The required demand, review, operating
capacity, and accessibility-user evidence is absent from this repository.
This record keeps the hypothesis explicit without manufacturing any of that
evidence.
