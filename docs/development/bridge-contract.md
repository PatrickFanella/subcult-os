# Superseded cross-repository bridge contract

Status: superseded by [ADR 0005](../adr/0005-subcult-os-platform-core.md). Retained so older links explain the prior design; do not implement a permanent OS-to-Subcults service bridge.

The useful constraints survive inside the unified platform:

- A private operator event may reference one public AT URI and observed CID.
- Public writes require local workspace authority, creator/profile delegation and current scoped OAuth.
- Public payloads use explicit allowlists and omit tickets, attendance, staffing, contacts, consent, finances, private notes and protected coordinates.
- Publication uses durable intent IDs, payload digests, CID preconditions, bounded retries and reconciliation.
- PDS timeout, projection lag, deletion, revocation and stale conflicts are visible states.
- Local event, ticket and door operation continues during protocol outages.

The implementation contract now belongs to the internal publication and discovery modules described in [architecture](architecture.md), with work ordered in the [backlog](backlog.md). Candidate logic and fixtures come through the [selective extraction inventory](extraction-inventory.md), not a network dependency on the old application.
