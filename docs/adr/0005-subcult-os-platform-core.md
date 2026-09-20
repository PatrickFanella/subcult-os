# ADR 0005: Make Subcult OS the Subcult.tv Platform Core

## Status

Accepted

## Context

Subcult OS and the older Subcults repository overlap in events, identity, public discovery, payments, and AT Protocol integration. Subcults contains valuable protocol, privacy, identity, migration, and conformance work, but it also contains years of experiments, retired features, duplicated routes, a very large historical test surface, and operational complexity that the intended Subcult.tv product does not need.

Importing Subcult OS into Subcults would make the older application's accumulated architecture the default. A cross-service bridge would preserve two account systems, two APIs, and two databases even though the desired product is one platform.

## Decision

Subcult OS is the receiving repository and the only target application architecture.

The platform will use:

- one Go API surface;
- one canonical user and session system;
- one PostgreSQL database with ordered migrations;
- bounded internal modules for identity, cultural records, event operations, ticketing, publication, discovery, consent, and delivery;
- creator PDS repositories as authority for approved public AT records; and
- private PostgreSQL records as authority for operational and sensitive data.

Subcults remains read-only source material during extraction. Capabilities may enter Subcult OS only through a reviewed extraction item with an explicit product need, minimal contract, provenance, dependency review, and focused acceptance tests. We will not merge the repositories, import Subcults' migration history, copy its complete test suite, or preserve an old feature merely because code exists.

Go remains the production backend language. TypeScript remains the frontend language and may provide an independent `@atproto/lex` conformance harness for the Go protocol boundary.

## Consequences

- The proposed cross-service bridge architecture is superseded.
- Subcult OS must gain ordered migrations before importing persistent capabilities.
- The existing Subcult OS `people` and `sessions` model must evolve into one platform identity system; accounts must not be merged by email equality alone.
- Subcults' accepted public-data and privacy decisions are evidence to evaluate, not blanket authority over the new repository.
- Useful Subcults implementations may be adapted or rewritten; unnecessary streaming, ranking, legacy route, deployment, and product surfaces stay behind.
- Every extracted capability must leave Subcult OS runnable and independently verifiable.
- Subcults remains intact until the replacement capabilities and any required data migration are qualified and separately authorized.
- Prototype API, schema, identifier and client compatibility is not required by default; see ADR 0006.
