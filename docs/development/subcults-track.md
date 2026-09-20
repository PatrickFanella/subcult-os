# Subcults source-extraction track

Subcults remains intact and read-only while Subcult OS absorbs selected capabilities. It is not the destination repository and is not a second service required by the target platform.

## Inspect, do not inherit

For each candidate, inspect the exact source revision, callers, database assumptions, dependencies, accepted ADRs, generated files, tests and operational prerequisites. Record the result in the [extraction inventory](extraction-inventory.md). Historical test volume is not evidence that a capability belongs in the new product.

## Prefer invariants over files

Extract the enduring invariant first: for example, OAuth state is single-use, session-family replay revokes descendants, a PDS timeout may follow a successful write, a public projection must match an observed CID, or protected coordinates cannot enter anonymous output. Write a small OS fixture for that invariant before adapting implementation.

## Candidate order

1. Lexicons and language-neutral conformance fixtures.
2. Identity/session threat cases and migration lessons.
3. Minimal AT syntax, resolver and OAuth behavior.
4. Publication intent, reconciliation and projection failure cases.
5. Minimal cultural model and privacy policies required by accepted journeys.
6. Consent/suppression vocabulary and negatives only when delivery work begins.

## Leave behind by default

Legacy route aliases, the large API composition, complete migrations/tests, old React application, streaming, ranking, alliances, social posts, general moderation, unused collections, duplicate provider integrations, deployment topology and provisioning remain excluded unless a later accepted journey supplies new evidence.

## Provenance

Every adapted file or fixture records the Subcults revision and source path. Rewrites record the source documents and invariants consulted. Do not edit or delete the old repository as part of extraction; retirement and archival require separate authorization after replacement qualification.
