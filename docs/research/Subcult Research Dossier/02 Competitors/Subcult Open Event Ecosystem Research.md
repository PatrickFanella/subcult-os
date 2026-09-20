---
type: competitor-and-ecosystem
status: draft
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Open Event Ecosystem Research

## OpenMeet

OpenMeet's February 2026 article describes managed AT identities, linking existing accounts through OAuth, and publishing public events/RSVPs with community event lexicons. It keeps private/unlisted events off the public PDS. The article explicitly says its account migration path had not yet been fully tested. These are first-party implementation claims, not a Subcult interoperability test. [OpenMeet identity/events](https://openmeet.net/your-identity-your-events).

**Implication:** “AT-native events” and ordinary onboarding with AT behind the scenes are already being pursued. Treat OpenMeet as both competitor and potential interoperability collaborator.

## Smoke Signal

Smoke Signal documents AT-native event activity and a history of community-lexicon work. Its Acudo article explores signed RSVP/ticketing integration, making it relevant prior art before proposing a new ticket-proof format. [Documentation](https://docs.smokesignal.events/), [Acudo article](https://blog.smokesignal.events/posts/3lwunvmen5k2b-introducing-acudo-bridging-atprotocol-and-event-ticketing-with-signed-rsvps).

**Implication:** shared calendar semantics and conformance examples are likely more useful than a competing bespoke event schema. Read existing implementation behavior before proposing extensions.

## Mobilizon

Mobilizon documents federated event/group organization in the ActivityPub ecosystem. It is a relevant open-source alternative, but ActivityPub federation is not automatic compatibility with AT repositories. [Mobilizon documentation](https://docs.mobilizon.org/about/).

**Implication:** portability and community control have precedents beyond AT. Any cross-protocol bridge needs explicit identity, visibility, correction and deletion semantics. A bridge is not simply a JSON shape conversion.

## Collaborate without depending on adoption

Build one read-only compatibility demonstration first: a public Subcult event should be interpretable outside our application, and a permitted external event should be displayable with provenance. Do not imply maintainers have agreed to collaboration.

Community schema repositories and discussion threads establish vocabulary and proposals, not universal platform support. [[Subcult Calendar Interoperability]] sets out the adapter and fixture work.

## Strategic result

The strongest ecosystem role is to contribute hard, shared cases from real cultural operations: multi-host events, tours, postponed dates, coarse location disclosure and private operational links that never leak. Maintain a usable product even if no upstream proposal is accepted.

Related: [[Subcult Upstream Contribution Portfolio]], [[Subcult AT Protocol Strategy]].

