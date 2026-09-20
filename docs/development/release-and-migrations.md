# Release and migration checklist
Use this checklist for a future authorized release. This task does not perform deployment.

## Before changing persistence
1. Capture source and schema versions.
2. Inventory every existing database read-only; classify it as empty, synthetic/reproducible, or retained real data.
3. Establish the clean platform schema through ordered migrations after DB-01.
4. Do not preserve prototype tables, identifiers or DTO shapes unless retained data creates a concrete migration requirement.
5. Test fresh install, replay and concurrent runners; test populated upgrade only for an explicitly retained dataset.
6. Prove failure stops incompatible startup.
7. Document recreation, rollback or forward-fix behavior before applying changes.

Subcults' migration ledger and readiness checks are reference patterns, not migrations to import. Build an OS-native ordered history from the actual OS alpha baseline.

## Before any legacy-data import
First establish that the data is real, authorized for retention and not reproducible. Then require an explicit candidate inventory, account-claim policy, deduplication rules, rights/source evidence, row-count invariants and dry-run report. Do not build a migration path for synthetic prototype data merely because it exists.
Do not auto-link by title/time alone: multi-act appearances, recurring events and reschedules can resemble duplicates. Queue ambiguous candidates for authorized review.

## Before enabling projection reads
Verify public allowlist, module/workspace access, resolver network bounds, safe rendering, freshness labels and missing/deleted record behavior.
Keep the flag default off until qualified. Choose actual flag names in implementation; none are introduced by these documents.

## Before enabling writes
Require accepted authority design; current scoped authorization; durable intents; stale-CID and lost-response tests; exact-CID reconciliation; rate limits and redacted logs.
Test revocation between intent approval and execution. Preserve unknown outcomes for reconciliation instead of clearing them.
Use synthetic disposable PDS records and explicit cleanup approval. Never replay unknown operations blindly.

## Before protected pilot
- [ ] Current full code gates and new identity/protocol/projection fixtures pass.
- [ ] Browser and physical-device flows verified on exact build.
- [ ] Public/private field review and independent security/privacy review complete.
- [ ] Clean backup restored independently; any explicitly retained external identifiers/history reconciled.
- [ ] PDS/stream disconnect, retry and projection rebuild rehearsed.
- [ ] Every adapted Subcults capability has provenance, dependency and intentionally-excluded behavior recorded.
- [ ] Named primary/secondary responder and incident contact exist.
- [ ] Privacy-safe telemetry, alerts, retention and failure runbook tested.
- [ ] Pilot participants, data scope and support expectations approved.

## Rollback
Disable new writes first. Retain durable intents and unknown remote outcomes. Keep local event operations running if their baseline remains healthy.
Before destructive cleanup, resolve exact targets through read-only checks and obtain explicit authorization. Avoid dropping populated identity, mapping or history tables until their retention classification is confirmed.
Public PDS writes are external state: disabling publication or reverting code does not undo them. Use reviewed compensating writes with fresh authorization and CID preconditions, or preserve the record and correct the projection as appropriate.
Once the new platform accepts non-synthetic tickets or other user data, restore is not a casual rollback. Reconcile post-backup transactions before any destructive recovery.

## Release evidence packet
Exact source/image/schema; configuration key names without values; checklist with pass/fail/not-run; browser/device record; backup/restore proof; known limitations; rollback target; owner and observation window.
A green health endpoint is necessary but not sufficient. No public release merely because a funding deadline arrived.
