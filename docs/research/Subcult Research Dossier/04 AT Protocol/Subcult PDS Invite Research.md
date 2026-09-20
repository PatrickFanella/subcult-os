---
type: contribution-research
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult PDS Invite Research

## Correct the premise first

The existing Subcults release/operations material flags strict invitation-expiry qualification. However, upstream AT/PDS code already includes an administrative disableInviteCodes procedure. Therefore “PDS has no invite revocation” is not a valid general novelty claim. [Disable-invite lexicon](https://github.com/bluesky-social/atproto/blob/main/lexicons/com/atproto/admin/disableInviteCodes.json).

Current account-manager implementation must be checked at the deployed version, not inferred from repository main. [PDS account manager](https://github.com/bluesky-social/atproto/blob/main/packages/pds/src/account-manager/account-manager.ts).

## Narrow problem statement

If the product promises an invitation expires after ten minutes, redemption must fail after that deadline even when a cleanup task has stopped. Administrative disablement and atomic expiration validation are related but different mechanisms.

The proposed question is whether the supported deployed PDS can enforce the product's expiry contract at redemption with correct concurrency behavior. That remains a test requirement, not a declared upstream bug.

## Test matrix

Create synthetic invites with known issue/expiry times. Test redemption before expiry, at the defined boundary, after expiry, after explicit disablement and under simultaneous attempts.

Stop any cleanup worker and repeat the after-expiry test. Restart services, simulate bounded clock differences under a documented time source, and verify that use limits cannot be exceeded through races.

Ensure retries distinguish already-used, expired, disabled and unknown outcomes without revealing sensitive account information. Record the exact binary/image revision and configuration.

## Contribution options

If current upstream already enforces the required contract, contribute documentation/tests or simply use it. If it lacks an expiry field or atomic check required by the use case, prepare a minimal design and regression test for maintainers.

Do not propose a custom PDS fork before establishing the actual gap and maintenance implications. Avoid promising production account provisioning until invitation, email, recovery and capacity gates pass.

Related: [[Subcult Repository Evidence]], [[Subcult Release Qualification]], [[Subcult Upstream Contribution Portfolio]].

