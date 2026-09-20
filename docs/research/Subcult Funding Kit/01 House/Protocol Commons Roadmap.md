# Protocol Commons Roadmap
These are proposed ecosystem contributions, not accepted AT Protocol changes. First collaborate on existing work; do not impose a new universal namespace.

| Package | Public benefit | Reference consumers | Proposed acceptance |
| --- | --- | --- | --- |
| Record lifecycle harness | Comparable create/update/delete, stale CID, replay and rebuild behavior | Subcults, Patchwork, dsa-proto | Two independently operated readers reproduce fixtures; deletion and stale-write failures visible |
| Organization authority experiments | Explicit scope, expiry, revocation and contested authority | dsa-proto, action-network | Negative tests for unauthorized and revoked writers; no claim of real-world official status |
| Public-resource provenance | Source, review date, expiry, correction and location precision | Patchwork, culture venues | Independent resource reader displays stale/conflicting evidence; no beneficiary information |
| Public/private publication contracts | Prevent accidental disclosure in approved artifacts | roberts-rules, organizer tools | Sensitive fixture fields rejected; public payload review receipts reproducible |
| Namespace migration guidance | Avoid silent dual-publishing and orphaned projections | Subcults, provisional civic schemas | Old/new reader fixtures; rollback plan and collision handling documented |
| Maintainer operations kit | Recovery, release provenance and incident handoff | All reference consumers | Another maintainer rebuilds a disposable environment from documented artifacts |

## What belongs upstream
Start with small reproducible bug reports, failing compatibility cases, documentation clarifications and SDK improvements against the relevant upstream repository. Follow its contribution rules and ask maintainers whether the proposal belongs there. A merge is the maintainer's decision. Core protocol amendments require broader design review; an application lexicon is not itself a core protocol feature.

Prioritize sync/recovery evidence and safe-publication tooling before speculative private-data or reputation extensions. AT's current overview describes non-public data as future work and cautions against bolting encrypted private content onto existing primitives. [Protocol boundaries](https://atproto.com/specs/atp). Use the published [Lexicon rules](https://atproto.com/guides/lexicon) and [OAuth specification](https://atproto.com/specs/oauth), not a new custom authentication protocol.

## Community process
Month 1: inventory existing schemas, location work and organization experiments; publish a problem statement and ask for comments. Month 2: narrow a fixture set with two willing external maintainers. Months 3–4: release reference tests and document incompatibilities. Months 5–6: independent reader and migration/recovery exercises. Months 7–12: maintenance, accessibility, review and handover rather than uncontrolled feature expansion.

The [AT Protocol Community Fund](https://atprotocol.dev/community-fund/) already lists related community infrastructure. Coordinate with location and resource projects before proposing duplicate foundational schemas. Hypercerts may be a partner for public action/provenance representations; a record is evidence of an assertion, not proof of social impact. [Hypercerts transition](https://hypercerts.org/blog/3mb6z2bku2c2z).

## Proposed $50,000 scoped package
$30,000 implementation, $8,000 independent review, $5,000 fixtures/documentation, $4,000 infrastructure and reproducibility, $3,000 administration. Six months with explicit capacity allocation; scope down if sponsor fees must come from this ceiling. This is a budget hypothesis, not a funder's award size or an additional $50,000 automatically added to the house budget.
Acceptance is reproducibility and external usability; production rollout and protocol adoption are excluded commitments.
