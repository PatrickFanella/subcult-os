# Membership Communications Proposal
## Decision and beneficiary
Organizations needing governed communication access. Membership revocation and identity handoff can fail across independently operated communication systems.

## Proposed intervention
Document and test entitlement/revocation across an AT-adjacent identity service and Matrix/OIDC.

## Starting evidence
Local-contract architecture; direct AT publication not evidenced and live authorized account gates remain open.
Source anchors: `dsa-signal`: backend/src/oidc, backend/src/identity, backend/src/entitlements, deploy/local and web/src/auth/matrix-oauth.ts.
This is a source inspection, not a fresh test pass. Consult [[DSA Evidence]], [[Patchwork Evidence]] and [[Subcults and OS Evidence]] for the relevant checkout. No real users or outcomes are inferred from code.

## Work and acceptance
Discovery: confirm the actual user need, rights, maintainers and funder scope before implementation.
Build: Document and test entitlement/revocation across an AT-adjacent identity service and Matrix/OIDC.
Acceptance: Two authorized test accounts in isolated topology; revocation and recovery evidence; independent review before security claims.
Handover: publish rights-cleared fixtures, installation and maintenance guidance, a limitations statement and an issue triage owner. Independent review must not be replaced by the founder's own checklist.

## Funding request
Proposed $20,000 scoped interoperability and independent security review, not a new messenger launch.
Budget at agreement: 60% implementation labor, 20% independent review, 10% fixtures/documentation, 5% essential infrastructure and 5% administration. These shares sum to 100%; adjust labor if review quotes or sponsor fees differ. They are a proposal design, not approved rates or funder eligibility.
Do not stack this package with another award for the same output. Match time allocation to [[Runway Model]] and [[Ninety Day Plan]] before signing.

## Data and governance
Private membership and communication metadata; no public membership graph.
E2EE does not hide service metadata or defeat a compromised browser origin; no secure-production claim.
Any organizational affiliation or pilot partner requires separate written permission. Source-code and data rights must be checked independently.

## Sustainability and alternatives
Infrastructure support contracts only after operational qualification.
Use Matrix ecosystem components rather than claiming to replace them; no affiliation with Signal.
A successful feasibility phase may recommend integration into an existing project instead of another standalone product. External reuse is a valid public-benefit outcome.

## Evaluation and stop conditions
Record the pre-work baseline and measurable failure cases before changing code. Two authorized test accounts in isolated topology; revocation and recovery evidence; independent review before security claims.
Stop at the current stage if licensing is unresolved, required partner authorization is absent, sensitive-data exposure cannot be controlled, or independent verification fails. Report failed acceptance results; do not remove the gate to maintain a fundraising narrative.

Presentation: [[Membership Communications Deck]]; speaker brief: [[Membership Communications Speaker Notes]].
