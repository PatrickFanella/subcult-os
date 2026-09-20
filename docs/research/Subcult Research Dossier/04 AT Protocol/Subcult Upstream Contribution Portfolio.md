---
type: contribution-plan
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Upstream Contribution Portfolio

These are proposed contributions, not submissions, accepted standards or agreed collaborations.

| Priority | Contribution | First destination | Demonstration of value |
| --- | --- | --- | --- |
| 1 | Calendar compatibility fixtures/adapter | Community lexicon and event-app maintainers | Independent reader handles real cultural edge cases |
| 2 | Projection recovery/conformance fixtures | Tap/tool maintainers where applicable | Rebuild converges without duplicate application effects |
| 3 | Multi-person authoring/recovery examples | Relevant auth/application discussions | Clear separation of OAuth, attribution and workspace policy |
| 4 | Synthetic Spaces collaboration cases | Permissioned-data proposal/implementers | Reproducible revocation, outage and recovery behavior |
| 5 | Private consent migration semantics | Interested application/provider tooling communities | Second importer preserves suppression and refuses unknown grants |
| 6 | Strict invite expiry tests or patch | PDS maintainers if an actual gap exists | Expired code fails without cleanup dependency |

## Contribution discipline

Start with existing specifications and implementation versions. Search issues and discussions, reproduce the need with synthetic data, and write the smallest useful artifact. Distinguish our application defect from an upstream gap.

The AT repository's contribution guidance should be checked before submission, including expectations for issue quality, tests and disclosure of AI assistance. No maintainer response or acceptance is guaranteed. [AT contribution guide](https://github.com/bluesky-social/atproto/blob/main/CONTRIBUTING.md).

## Effort budget

Initially cap protocol work at a proposed 10–15% of engineering effort. That is a planning choice, not an industry benchmark. Prefer work that immediately improves the product's correctness or interoperability.

Raise the cap only when an external consumer, funded collaboration or demonstrated product dependency warrants it. Avoid turning the company into an unfunded standards project while customer qualification remains unresolved.

## Evidence required for each item

Record problem, prior art, chosen owner/community, current version, smallest reproduction, proposed semantics, test artifacts, second-consumer feedback and fallback. A compelling design document without a consumer is not interoperability.

Do not submit private customer examples, contact lists, venue safety information or deployment credentials. Synthetic fixtures should preserve the hard semantics without carrying real identities.

## Fallbacks

A community schema extension can remain a documented local adapter. A rejected core change may be avoidable with existing primitives. Spaces can remain experimental while Studio uses private storage. An unavailable ticketing integration can remain a link with manual closeout.

The business must function under these fallbacks. That constraint keeps contribution choices focused and makes upstream work more credible.

Related: [[Subcult Calendar Interoperability]], [[Subcult Sync and Projection Contributions]], [[Subcult Decisions and Open Questions]].

