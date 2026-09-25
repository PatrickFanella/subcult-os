---
type: engineering-research
status: synthetic-fixture
created: 2026-09-25
issue: OFFLINE-01
---

# Disconnected door-operation experiment

## Scope and boundary

This is a synthetic, offline decision exercise for OFFLINE-01. It introduces
no application route, migration, ticket cache, browser storage, account,
network service, provider call, deployment, or production admission path. The
existing server-authoritative door flow remains the only implemented admission
flow.

The experiment uses literal synthetic ticket codes and names. It does not use
real people, tickets, contact details, payment data, devices, scanners, or
radio/network connections. A passing fixture is not proof of physical-device
operation, OS storage behavior, scanner behavior, network partition handling,
or readiness under door pressure.

## Fixture model

`backend/internal/offlinelab` defines a small server-authoritative merge model.
A snapshot includes only a synthetic event ID, snapshot ID, roster revision,
fixture expiry timestamp, ticket code, display name, eligibility, and checked-in
state. It intentionally omits email, payment state, amount, currency, contact
fields, and any product data-store integration.

The fixture TTL is `15 minutes`. That is a test input chosen to exercise expiry;
it is not a proposed operational policy or a retention decision. Any pilot
would need an explicit owner-approved expiry and a fresh online authorization
before a snapshot is issued.

A client can create only a provisional local operation while the fixture
snapshot is unexpired and the ticket is eligible. The local operation does not
admit anyone. At reconnect, the synthetic server is authoritative:

| Condition | Fixture merge result | Operational meaning still unresolved |
| --- | --- | --- |
| First eligible operation | `accepted` | A future pilot must define its visible door confirmation. |
| Same ticket from another partitioned client | `duplicate_check_in` | The losing action needs a clear conflict/reconciliation view. |
| Retry of an already received operation | `duplicate_operation` | The operation ID prevents a second server mutation. |
| Snapshot has expired | `expired_snapshot` | Decide the event-specific refresh and fallback procedure. |
| Snapshot revision is stale | `stale_snapshot` | Decide review versus a new online roster. |
| Client was revoked before reconnect | `revoked_client_review` | Do not claim that offline revocation prevented a past physical entry. |

The model deliberately holds revoked-client work for review. It does not turn
a revocation into a retroactive physical-door decision.

## Roster, revocation, and device loss

Current workspace authority remains online and server-authoritative. A
member's expiry or revocation can stop a future server request and prevent a
new snapshot from being issued. It cannot retract a roster already copied to a
disconnected client. This distinction matches the existing authority model's
active-membership checks and should remain explicit in any future operator
interface.

For a lost device, a future procedure must record the device,
snapshot/revision, discovery time, responsible operator, and action taken;
revoke the relevant membership or credential online; stop using the device;
and move the door to a qualified online, provider, or manual process. Whether
a particular device cache was erased needs device-specific evidence. This
repository does not have that evidence.

A manual fallback is a separate accountable record, not an automatically
replayed queue. It needs a responsible operator, numbered entries, timestamps,
and server reconciliation before closeout. The synthetic harness records that
these fields are required; it does not implement the form, storage, or policy.

## Separate-client fixture

Run the harness after normal dependencies are available:

```bash
node scripts/offline-door-experiment.mjs
```

The harness creates two separately launched Node processes. Each receives the
same fixture snapshot, writes only in a distinct temporary storage path, and
records a provisional scan of `SYNTH-ONE` while partitioned. The parent fixture
reconnects client B before client A, then retries B's operation as if a response
had been lost. Expected merge results are `accepted`, `duplicate_check_in`, and
`duplicate_operation`. It also checks expired and revoked fixture operations
and reports a manual-fallback record shape.

The focused model tests are:

```bash
cd backend && go test ./internal/offlinelab -count=1
```

They cover the same conflict, retry, expiry, revocation, eligibility, and stale
revision cases at the model boundary. The harness adds process and
storage-isolation fixtures; it is not an emulator of a real client.

## Required gate before an offline product slice

Do not enable disconnected admission from these results. A later pilot needs,
at minimum:

1. An explicit owner decision for snapshot expiry, permitted fields, refresh,
   revocation behavior, and reconciliation ownership.
2. Two independently operated physical devices using approved synthetic data,
   with documented local persistence and separate credentials.
3. Observed airplane-mode or equivalent partition/reconnect, app restart,
   duplicate scan, stale roster, revocation, lost-device, and manual-fallback
   journeys.
4. Exact served revision, timestamps, operation IDs, snapshot revisions,
   expected and observed merge results, and retained failed evidence.
5. An accountable event operator review of fallback and closeout reconciliation.

Until then, use the current online server-authoritative door flow or a ticket
provider's qualified door tooling. This experiment completes only the research
slice; it does not complete a physical-device or production-offline gate.
