# Subcult OS platform-core track

Subcult OS is the only target application. Preserve its working event lifecycle while adding foundations in independently verifiable slices.

## Immediate foundations

API-01 established an explicit public/private projection invariant and remains useful even though its prototype route/type shape may change. DB-01 replaces the embedded alpha schema with a clean ordered, ledgered, failure-safe schema before identity or AT state is added. INV-01 prevents old Subcults architecture from entering by accident.

## Module extraction

Split new code by domain boundary instead of enlarging `backend/internal/app` indefinitely. Begin with identity and AT adapters behind narrow interfaces, then cultural records, projection, and publication. Update routes and clients together; add compatibility code only for a verified consumer or retained dataset, with a named removal condition.

## Identity

Design the canonical account and session model directly. Do not preserve prototype passwords, sessions, person IDs or login payloads unless an inventory identifies real accounts that require migration. For any such migration, require fresh proof for claims and keep platform, workspace, creator, and AT authority separate.

## Cultural and operational events

Do not overload the existing operational event row with every public cultural field or protocol state. Add a minimal cultural occurrence model and an explicit private relationship. Local tickets, door, settlement and archive continue when AT services are unavailable.

## Protocol

Use pinned Indigo components through an adapter and shared Go/TypeScript fixtures. Import no general protocol implementation that upstream already owns. Public writes arrive only after read/projection behavior and authority have passed their gates.

## Client cutover

Change API, web and mobile contracts in the same slice. No legacy route aliases are required by default. Real browser and device checks are separate acceptance evidence; builds and unit tests do not establish session, deep-link, camera or secure-storage behavior.
