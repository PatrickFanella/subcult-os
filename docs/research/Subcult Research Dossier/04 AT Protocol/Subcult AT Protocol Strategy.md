---
type: protocol-strategy
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult AT Protocol Strategy

## Use the protocol for specific public benefits

AT offers portable account identity, repository-based public data and application-defined schemas. The proposal uses those properties for profiles, places, scenes, occurrences, appearances and tours that are intentionally public. Other applications still need compatible schemas and indexing/UI support. Custom records do not automatically become Bluesky posts. [AT overview](https://atproto.com/guides/overview), [Lexicon specification](https://atproto.com/specs/lexicon).

Do not sell AT as a payment network, marketing-permission transfer system or complete organization-access model. Those are separate application responsibilities.

## Architecture stance

Public PDS records can be canonical while the app maintains a rebuildable projection for maps/search. Private drafts and operations remain in the application database. A publish action should show accepted/pending/conflicted states honestly.

The existing Subcults namespace is `tv.subcult.*`; legacy intake and community schemas need an explicit compatibility policy. Do not silently publish duplicate occurrences into every namespace.

Account migration is a protocol capability requiring an operational workflow, including repository and blob handling. Do not claim that the ability to change a DID service endpoint proves a complete tested customer migration. [AT migration guide](https://atproto.com/guides/account-migration).

## Public integrity is not social truth

A valid signed record establishes provenance/integrity within its trust model. It does not prove that an artist agreed to perform or that a venue is safe. Preserve corrections and moderation routes for disputed claims.

Labels are an existing ecosystem mechanism for metadata, including moderation uses. Investigate reuse before inventing incompatible public annotations, while keeping private incident information out of public labels. [Label specification](https://atproto.com/specs/label).

## Contribution philosophy

Begin with a real application need, identify existing specifications and maintainers, and produce minimal interoperable evidence. Prefer a fixture, bug reproduction, adapter or documentation correction over a sweeping new protocol layer.

Every proposed contribution needs a second consumer and a fallback if upstream declines it. The business should not depend on unilateral standard-setting authority.

## Near-term priorities

1. Calendar semantics and privacy-safe event compatibility.
2. Projection recovery tests using existing sync infrastructure.
3. Narrow OAuth/organization UX and recovery evidence.
4. Synthetic permissioned-data and invitation-expiry experiments.

Details: [[Subcult Calendar Interoperability]], [[Subcult Upstream Contribution Portfolio]], [[Subcult Spaces Research]].

