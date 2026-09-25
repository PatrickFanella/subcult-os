---
type: protocol-research
status: exploratory
created: 2026-09-24
research_as_of: 2026-09-24
tags:
  - subcult-research
  - spaces
---

# Spaces private-collaboration access-model experiment

## Scope

This is an offline, synthetic decision-engineering exercise. It contains no provider connection, protocol request, account, personal data, production route, migration, or deployment. The in-memory model in `backend/internal/spaceslab` is one implementation of a deliberately small policy state machine. It is not an AT Protocol Spaces client, a protocol contract test, or evidence that two real implementations interoperate.

The fixture uses synthetic identifiers and literal fixture content only. Its purpose is to make the research questions around temporary collaboration explicit before any alpha sandbox work.

## Sources checked on 2026-09-24

The [Atproto Spaces alpha announcement](https://atproto.com/blog/atproto-spaces-alpha) says Spaces provides access control for non-public data, not confidentiality; it also says the alpha has breaking changes and must not be used in production. The announcement describes direct application sync from PDS hosts and cautions against uploading sensitive data.

The current [permissioned-data proposal](https://github.com/bluesky-social/proposals/blob/main/0016-permissioned-data/README.md) still labels itself a proposal rather than a final specification. It describes a space as an authorization and sync boundary, with credentials issued by a space authority and bound to the acting application. The proposal leaves the authority's membership decision outside the protocol. These sources inform the questions below; neither is treated as a stable product contract.

The existing [Subcult Spaces research](/docs/research/Subcult%20Research%20Dossier/04%20AT%20Protocol/Subcult%20Spaces%20Research.md) remains the wider research context.

## Executable synthetic cases

Run the isolated experiment with:

```bash
cd backend && go test ./internal/spaceslab -count=1
```

The model records only a synthetic principal, one of three synthetic roles, an expiry, a removal marker, and a local cache of records that were already read. It exercises the following hypotheses:

| Case | Modelled outcome | Boundary of the result |
| --- | --- | --- |
| Temporary crew membership | A crew read is allowed before expiry and denied at expiry. | Does not test credential issuance, clock handling, or an external host. |
| Removal and replay | Removal denies future reads and rejects an ordinary replayed grant. | It is an application policy choice, not a claim about Spaces revocation semantics. |
| Application withdrawal | A withdrawn application grants no access. Withdrawal also does not silently revoke an independently granted membership. | It does not model Subcult's application data or any protocol record. |
| Recovery | Restoring access requires a separate, explicit synthetic recovery decision. | It does not establish a real recovery authority or operational process. |
| Finance separation | A crew principal is denied a synthetic finance-scope record; a finance principal can read it. | It is a research fixture, not a proposed mapping of Subcult roles into a Space. |
| Residual cache access | Removal denies a new read, but a record copied earlier remains locally available until the application clears its cache. | It does not measure a device, browser, PDS, syncer, backup, or another application's cache. |

The cache case keeps two statements distinct: revocation can prevent a future server-side read, while it cannot retract a copy already delivered to a client. Any future user-facing design must explain that distinction and provide bounded local-data handling rather than promising retrospective erasure.

## Result and decision

The offline tests provide a repeatable way to discuss temporary membership, scoped access, withdrawal, explicit recovery, and copied-data limits without using real people or records. They do not validate Spaces protocol behavior, security properties, interoperability, or a consumer need.

**Decision: no-go for a product integration.** Before reopening that decision, obtain demonstrated demand, current stable semantics, a scoped threat assessment, an independently reviewed recovery process, and observed compatible behavior from at least two genuinely independent Spaces implementations using only approved synthetic test data. Run a separate experiment against those implementations and record network, credential, cache, outage, migration, and revocation evidence. The present package cannot substitute for any of those gates.
