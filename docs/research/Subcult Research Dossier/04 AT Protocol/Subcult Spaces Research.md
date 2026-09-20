---
type: protocol-research
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Spaces Research

## Current status

The August 2026 AT Spaces announcement explicitly describes an alpha for permissioned data and warns against sensitive production use before security review and further work. Restricted repository access is not end-to-end encryption, and its architecture differs from public relay distribution. [Spaces alpha announcement](https://atproto.com/blog/atproto-spaces-alpha).

Read the evolving proposal for current semantics before each experiment; an alpha announcement is not a stable contract. [Permissioned-data proposal](https://github.com/bluesky-social/proposals/blob/main/0016-permissioned-data/README.md).

## Why it could matter

Cultural teams need temporary shared contexts: a tour crew, a one-night production team, a volunteer group or a collaborative draft. Portable permissioned records could eventually support these contexts across applications.

That makes Subcult a useful source of concrete test cases. It does not justify moving customer contact lists, protected addresses or financial records into experimental infrastructure.

## Bounded experiment

Create synthetic participants and one temporary crew Space. Test invitation, membership update, departure, authority outage, PDS outage, restoration, application access and attempted unauthorized reads.

Distinguish server-enforced future access revocation from erasing content a member already downloaded. Define who can change membership, how that authority recovers, and how another app knows the current policy.

Record what happens when a member's account migrates, a device retains cached data, or the application loses permission. Any public reference to a Space must avoid exposing sensitive existence or relationships unintentionally.

## Useful contribution

A threat-oriented fixture set and clear UX/recovery documentation could benefit other collaboration apps. File precise issues against observed alpha behavior rather than demanding that a young proposal solve all organizational governance.

Potential research questions include portable authority, cache expectations, export, membership history and disaster recovery. These are questions, not claims of missing implementation.

## Gate to production consideration

Require stable relevant semantics, independent security review evidence, scoped application threat assessment, recovery rehearsal, access-revocation tests, migration/backup behavior and a clear user explanation. Compare the result against the private-database baseline.

Until then, Studio stays on conventional private storage. Spaces is an isolated research track with a fixed effort cap.

Related: [[Subcult Privacy and Consent]], [[Subcult Organization and OAuth Research]], [[Subcult Upstream Contribution Portfolio]].

