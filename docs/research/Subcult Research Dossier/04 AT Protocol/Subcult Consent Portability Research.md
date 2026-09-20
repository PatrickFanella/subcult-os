---
type: contribution-research
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Consent Portability Research

## The opportunity is narrower than “portable audiences”

Exportable contacts are not the same as transferable permission. A useful private migration bundle could preserve sender, channel, purpose, exact disclosure/version, capture evidence, verification state, withdrawal and suppression history.

The goal would be faithful migration for a legitimate continuing sender, not automatic permission for a new artist, venue, sponsor or platform to message everyone.

Provider ecosystems already implement consent controls. Twilio documents a Consent Management API, including opt-in/opt-out records and integration behavior. Investigate existing semantics before inventing a parallel universal ledger. [Twilio Consent Management API](https://www.twilio.com/docs/messaging/features/consent-api).

## Proposed private interchange experiment

Use synthetic contacts. Export a small audience with both grants and revocations. Import it into a second test application without activating delivery. Compare decisions for every sender/purpose/channel combination.

A revoked contact must remain suppressed. Unknown or incomplete evidence must not become a positive grant. Changes of controller/sender or purpose should trigger review rather than silent inheritance.

Keep the package encrypted/access-controlled in transit and at rest according to the selected implementation. Do not place it in public PDS records, content hashes or public audit trails.

## Scope and non-goals

This is initially an application/tooling convention, not necessarily a core AT change. The shared value is preserving evidence and negative permissions across migrations.

A protocol cannot declare consent legally valid across jurisdictions. A successful file import cannot prove a campaign is lawful or acceptable to a provider. Legal, provider and user-expectation checks remain separate.

## Required fields to evaluate

Sender identity; contact type and normalized address; purpose; disclosure text/version; capture source; time; verification method; grant/revocation history; suppression reason; provenance; export authority; retention/deletion restrictions.

Minimize fields to what the actual use case requires. More metadata is not inherently better if it exposes sensitive relationship history.

## Contribution gate

Find another legitimate application willing to test synthetic import/export. Document mismatches and refusal behavior. Propose shared semantics only after the exercise demonstrates value.

Related: [[Subcult Privacy and Consent]], [[Subcult Audience Platform Research]], [[Subcult Upstream Contribution Portfolio]].

