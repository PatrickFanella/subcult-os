---
type: product-and-safety-proposal
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Privacy and Consent

## Default classification

| Data | Initial treatment | Reason |
| --- | --- | --- |
| Approved artist profile, public event description | Public AT record eligible | Intentionally shareable cultural material |
| Chosen public credits | Separate approval before publication | Participation alone does not imply publication |
| Email, phone, consent evidence, suppression | Private application data | Contact and permission are contextual |
| Crew roster, applications, private notes | Workspace/event-scoped access | Avoid exposing people and internal judgments |
| Exact protected address, location grant | Private access-controlled service | Coarse public discovery must not reveal it |
| Orders, payment references, settlement | Private authoritative commerce/operations | Financial integrity and personal data |
| Public RSVP | Explicit opt-in only | Attendance can reveal sensitive associations |

Do not publish hashes of contact details or consent receipts as a supposed privacy shortcut. Such values can remain linkable and do not make a marketing permission public-domain data.

## Relationship distinctions

Following an Act, joining a Scene, saving a date, reserving a place, applying for a role and subscribing to promotions are separate relationships. The interface must explain which action is happening and who receives information.

Track consent by sender, channel, purpose and contact point, with wording/version, capture source, timestamp, verification and revocation. Re-check eligibility immediately before delivery, not just when a campaign is scheduled.

Twilio's messaging policy requires consent, sender identification and a withdrawal mechanism, subject to applicable rules; provider approval is not a substitute for legal review. [Twilio Messaging Policy](https://www.twilio.com/en-us/legal/messaging-policy).

## Location and sensitive participation

Use a deliberately coarse area for protected gatherings. Release precise directions only through an authorized, revocable workflow. Do not derive and expose a person's home from repeated participation or their device location.

Public deletion cannot guarantee removal of copies already obtained by others. Explain visibility before publication rather than promising retroactive secrecy. The safest way to protect private information is not to publish it.

## Controls required before pilot

Provide opt-out in each supported channel, an understandable preference center, staff access revocation, data export boundaries, retention policy and an incident procedure. Log why a delivery was authorized without exposing contact data in general-purpose logs.

Use synthetic data for development and protocol experiments. Customer interview notes should not be placed in a broadly shared research vault with names/contact details unless consent and access rules support it.

## Open legal work

This note proposes product safeguards, not a legal compliance determination. Before paid messaging or commerce, obtain jurisdiction-specific review of marketing consent, transactional notices, minors, retention, processor/controller roles and consumer obligations.

Related: [[Subcult Consent Portability Research]], [[Subcult Spaces Research]], [[Subcult Risk Register]].
