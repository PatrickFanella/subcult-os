---
type: contribution-proposal
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Calendar Interoperability

## Existing work first

OpenMeet publishes using community event lexicons; Smoke Signal is a relevant implementation and community participant. Community discussions include event media and interoperability details. This is prior art, not evidence that all event apps implement the same version. [OpenMeet public events](https://openmeet.net/your-identity-your-events), [community discussion](https://discourse.lexicon.community/t/event-header-images/39), [Smoke Signal docs](https://docs.smokesignal.events/).

Subcults currently has domain-rich `tv.subcult.*` records. The decision is how to map them without fragmenting occurrence identity or losing music-specific relationships.

## Proposed artifact

Create a versioned, openly licensed compatibility fixture pack with a mapping document and one reference adapter. It should explain canonical occurrence IDs, source revision, author/controller, local times, changes, public host relationships and intentionally omitted private fields.

The pack should be useful to calendars that know nothing about Subcult's private Studio. It should not require a private Subcult API to interpret a public event.

## Required cases

| Fixture | Expected behavior |
| --- | --- |
| Local show with two hosts | One occurrence; public credits do not grant editing rights |
| Touring Act in another city | Discovery uses occurrence location, not home territory |
| Multi-day festival | Explicit distinction between parent gathering and individual occurrences |
| DST transition | Unambiguous timestamp plus display-zone policy |
| Postponement | Current status and correction retained without duplicate attendance |
| Cancellation | Independent reader reflects cancellation instead of a stale active listing |
| Protected place | Coarse location only; precise coordinates never enter public record |
| Deleted/corrected record | Reader processes absence/correction under documented policy |
| External ticket link | No embedded private entitlement or order information |

## Namespace decision

Evaluate three options: adopt community records as the public calendar surface with domain extensions; keep native records with an explicit adapter; or propose a compatible extension upstream. Document lossiness and authority for each.

Do not dual-publish by default. If more than one representation is necessary, specify canonical identity, link relationships, deduplication and correction propagation. Otherwise one event can appear twice and drift independently.

## Acceptance and contribution path

An independent implementation must consume the fixture and show the expected occurrence/time/status without private data. A unit test in our own serializer is insufficient. Publish no proposal until current schema versions and maintainer preferences are checked.

This contribution belongs first with relevant schema/application communities; only a demonstrated core-protocol gap should become a core AT proposal.

Related: [[Subcult Domain and Authority Model]], [[Subcult Open Event Ecosystem Research]], [[Subcult Upstream Contribution Portfolio]].

