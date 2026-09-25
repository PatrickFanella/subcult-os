# Consent portability experiment — 2026-09-24

Status: completed offline engineering experiment for issue #67. This report
records a synthetic contract exercise only. It does not establish a production
interchange, a maintained adapter, a legal conclusion, provider acceptance, or
permission for a different sender to contact anyone.

## Question and boundary

The question was whether a private bundle could preserve consent evidence and
negative states across an application handoff without treating a contact export
as new permission. The experiment uses no database rows, API routes, delivery
workers, providers, public AT records, real addresses, credentials, or message
content.

The source application already treats announcement permission as an exact
workspace, channel, recipient and purpose grant. Durable suppression wins over
every grant. See [`../development/consent.md`](../development/consent.md) and
[`../development/data-boundaries.md`](../development/data-boundaries.md).

## Fixture contract

`backend/internal/consentport` defines `subcult.consent-port/v1`, a bounded
JSON research format. The committed fixture uses only `.example.test`
addresses and fixed synthetic identifiers/times.

Each record carries only the fields needed to evaluate evidence:

| Need | Contract field |
| --- | --- |
| Sender and continuing controller | `sender.id`, `sender.controllerId` |
| Channel, purpose, profile and scope | `channel`, `purpose`, `profile`, `scope` |
| Recipient | `recipient.kind`, `recipient.normalizedAddress` |
| Disclosure | `disclosure.version`, `disclosure.text`, `disclosure.sha256` |
| Capture and verification | `capture.source`, `capture.grantedAt`, `verification.method`, `verification.verifiedAt` |
| Withdrawal and suppression provenance | `withdrawal` / `suppression` timestamp, reason and provenance |
| Stable source provenance | bundle `exporter.id`, `exporter.authorityRef`, record `sourceGrantRef` |
| Retention restriction | `retention.restriction` |

The format is explicitly `transferMode: "evidence_only"`. It excludes raw
confirmation or withdrawal tokens, credentials, operator/person identifiers,
message bodies, ticket/contact/membership data and public URLs. The decoder
limits an entire bundle to 256 KiB, limits it to 64 records, rejects unknown
JSON fields and rejects multiple JSON values. The source contract computes the
canonical lowercase SHA-256 of `disclosure.text` and requires it to match
`disclosure.sha256`.

## Second receiver experiment

The test receiver is independently written in the external test package. It
decodes JSON into separate receiver structs and uses a local allowlist for the
known synthetic exporter authority, source-grant provenance registry,
continuing sender/controller, email announcement channel/purpose, `newsletter`
profile, `monthly-newsletter` scope and disclosure text/version/digest. The
registry binds each known synthetic source-grant reference to its immutable
sender, controller, recipient, channel, purpose, scope and profile, so a
record cannot reuse a known reference for another audience identity. It does
not call the source contract validator or any production API/DB code. It
parses the bundle, capture, verification and negative-evidence timestamps
itself as RFC3339 values.

The receiver accepts a matching record as **evidence** only. Its `CanContact`
result remains false for every imported row, including verified source grants.
It refuses an unknown exporter or source-grant reference, a record that binds a
known source-grant reference to a different sender/controller/recipient/
channel/purpose/scope/profile, an unknown disclosure, incomplete verification,
a duplicate source reference, and a transfer mode that asserts permission. The
receiver validates a whole bundle into a
temporary state and swaps it only after every record passes, so a rejected
bundle leaves no partial import.

Withdrawal and suppression are terminal in this experiment. A withdrawal
delta changes an earlier active evidence record to withdrawn. Replaying the
older active bundle cannot reactivate it. Suppression has precedence if both
negative states are present. These are preservation rules for evidence, not a
delivery decision for a receiving product.

## Results and decision

The executable fixture demonstrates that a small private representation can
carry sender/channel/purpose/scope/verification/disclosure/suppression
provenance, that a separately implemented receiver can refuse untrusted or
incomplete material without partial state, and that negative evidence survives
replay. It also demonstrates the required distinction: evidence transfer does
not create contact permission.

Engineering decision: **no-go for a maintained interoperability adapter at
this time**. No identified real consumer, continuing-sender migration use case,
transport/key-management design, or external compatibility exercise supports
one. Keep this fixture and test as research evidence. Reconsider only for a
specific private migration between identified controllers, with its own
privacy/security, provider and operational review. A future product would need
separate decisions for authenticated export authority, encrypted storage and
transport, recipient identity matching, retention/deletion, re-consent, local
send authorization, and real receiver compatibility.

## Verification

Run the focused, offline experiment:

```bash
cd backend && go test ./internal/consentport -count=1
```

For a change that enters the repository, also run the repository's required
verification in its disposable development environment:

```bash
make verify && make test-db
```

Result recorded 2026-09-24: the combined focused race check, `make verify`
and `make test-db` completed with exit status 0. The run used the disposable
development-environment database; it did not contact a provider or import
non-synthetic audience data.

After the independent review corrections, the focused package race tests and
`make verify && make test-db` passed again in the installed isolated environment
on 2026-09-24. The corrected disclosure digest and independent timestamp refusal
cases are included in that result.
