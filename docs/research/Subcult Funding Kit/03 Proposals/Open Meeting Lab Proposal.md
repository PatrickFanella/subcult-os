# Open Meeting Lab Proposal
## Decision and beneficiary
Community facilitators and governance-software builders. Meeting procedure and public outcomes need traceability without disclosing deliberation or ballots.

## Proposed intervention
Develop the event-sourced simulator and reusable approved aggregate-publication contracts.

## Starting evidence
Implemented meeting simulator; experimental AT adapter; explicitly no production PDS integration.
Source anchors: `roberts-rules`: packages/meeting-domain/src, packages/meeting-publication and packages/atproto-publication/src/atproto-repository.ts.
This is a source inspection, not a fresh test pass. Consult [[DSA Evidence]], [[Patchwork Evidence]] and [[Subcults and OS Evidence]] for the relevant checkout. No real users or outcomes are inferred from code.

## Work and acceptance
Discovery: confirm the actual user need, rights, maintainers and funder scope before implementation.
Build: Develop the event-sourced simulator and reusable approved aggregate-publication contracts.
Acceptance: Replay deterministic fixtures; prove private fields cannot enter published artifacts; accessible facilitator rehearsal.
Handover: publish rights-cleared fixtures, installation and maintenance guidance, a limitations statement and an issue triage owner. Independent review must not be replaced by the founder's own checklist.

## Funding request
Proposed $25,000 four-month public/private governance R&D package, not an election system.
Budget at agreement: 60% implementation labor, 20% independent review, 10% fixtures/documentation, 5% essential infrastructure and 5% administration. These shares sum to 100%; adjust labor if review quotes or sponsor fees differ. They are a proposal design, not approved rates or funder eligibility.
Do not stack this package with another award for the same output. Match time allocation to [[Runway Model]] and [[Ninety Day Plan]] before signing.

## Data and governance
Public approved aggregate outcomes; private attendance, rosters, deliberation, credentials and ballots.
No secret ballots, binding voting, legal procedure certification or copied Robert's Rules text.
Any organizational affiliation or pilot partner requires separate written permission. Source-code and data rights must be checked independently.

## Sustainability and alternatives
Training, support and grant-backed maintenance; rights reviewed for all instructional content.
Complement facilitation tools with reproducibility and explicit publication control.
A successful feasibility phase may recommend integration into an existing project instead of another standalone product. External reuse is a valid public-benefit outcome.

## Evaluation and stop conditions
Record the pre-work baseline and measurable failure cases before changing code. Replay deterministic fixtures; prove private fields cannot enter published artifacts; accessible facilitator rehearsal.
Stop at the current stage if licensing is unresolved, required partner authorization is absent, sensitive-data exposure cannot be controlled, or independent verification fails. Report failed acceptance results; do not remove the gate to maintain a fundraising narrative.

Presentation: [[Open Meeting Lab Deck]]; speaker brief: [[Open Meeting Lab Speaker Notes]].
