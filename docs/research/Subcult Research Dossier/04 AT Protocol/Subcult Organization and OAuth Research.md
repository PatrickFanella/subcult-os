---
type: protocol-research
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Organization and OAuth Research

## What exists

AT's OAuth profile and permission specification already provide application authorization and collection/action scoping. Start from those capabilities rather than proposing generic granular permissions as though none exist. [OAuth specification](https://atproto.com/specs/oauth), [permission specification](https://atproto.com/specs/permission).

Those mechanisms do not by themselves define Subcult's complete organizational policy: who owns a venue, who may invite crew, which artist can approve a credit, and who recovers control after an administrator leaves.

## Distinctions

Authentication answers which account is acting. OAuth authorizes an application to act within granted scope. Workspace policy determines whether this person may perform this event operation. Public attribution states a relationship but should not confer private permission.

An application authorized to write a collection must still apply product-level checks. A public “manager” assertion is not safe enough to grant settlement access.

## Initial design

Keep private roles in the workspace service. Use named accounts, event-scoped invitations and explicit revocation. Avoid shared credentials. Give crew only needed capabilities and define an owner-recovery procedure before real usage.

For public organization profiles, document who publishes and how control is delegated or handed over. Do not conflate a collective's cultural identity with a legal merchant or assume a scene DID resolves its governance.

## Contribution opportunity

Document realistic multi-person authoring and recovery cases, then compare them to current ecosystem work. A minimal interoperable example could distinguish public authorship attribution, application permissions and private role enforcement.

Potential upstream work should follow a demonstrated gap: for example a reproducible authorization UX ambiguity or a missing standardized representation after existing options are evaluated. Do not invent a universal organization standard solely for our data model.

## Acceptance tests

Removing a worker revokes private access immediately within defined service semantics. Public credit remains if appropriate. A team member cannot elevate their role through a public record. A legitimate owner can recover access without rewriting historical event authorship. Application revocation is handled without destroying unrelated account data.

Related: [[Subcult Domain and Authority Model]], [[Subcult Spaces Research]], [[Subcult Release Qualification]].

