# Patchwork evidence for the Subcult.tv AT Protocol house kit

**Review date:** 2026-09-19  
**Scope:** read-only repository evidence; no runtime, credentialed, or production test was run.  
**Verdict:** Patchwork is a credible open-source, AT-native *pre-alpha foundation* for a future mutual-aid/public-resource vertical, but it must be pitched as an R&D and protected-pilot candidate—not a public mutual-aid service, proven production deployment, or demonstrated social-impact operation.

## Exact checkout and lineage

| Checkout | Revision | Date / branch | What it represents |
| --- | --- | --- | --- |
| `/home/onnwee/Work/subcult/patchwork` | `29bb643e8eab33d42b5fe7cd30001e350ee0f8a2` | 2026-09-19, `codex/documentation-checkpoint-20260919` | Canonical working checkout reviewed. It has user-owned untracked `.playwright-cli/` and `output/` directories; neither was inspected or changed. |
| `/home/onnwee/Work/subcult/patchwork-mobile-handoff` | `480535143e1176737a6fd075c2af7c2b3c712d12` | 2026-09-13, `codex/chicago-public-resources` | Separate, older handoff; its last commit adds private listing corrections and independent review. Its README frames requests at five-digit-US-ZIP precision and verified public-resource pins. Do not call it the newer source of truth. |
| `/home/onnwee/.t3/worktrees/patchwork/t3code-2b92787d` | `79bb453bc9b0360090adbd3abfb5e665a92c0a47` | `t3code/source-refresh-chicago-data` | Separate newer routing/data-refresh worktree; not audited here. |

The canonical checkout’s diff from the mobile handoff is unusually large (500 files changed, including deleted legacy seed/catalog paths). That is a material migration/documentation-risk signal. The current checkout’s own README and state matrix—not a historical release note or the handoff—should anchor fundraising claims.

## License and fundability implications

Both checked-out copies contain the GNU GPL v3 license and state **GPL-3.0-or-later** in their READMEs (`patchwork/README.md:185-187`; `patchwork-mobile-handoff/README.md:85-87`). This is a strong fit for a public-goods/open-source grant narrative: the platform and its improvements remain copyleft when conveyed. It is not, by itself, a commercial model. Any proposal should pair it with a separately reviewed sustainability model (hosted operations, implementation/support, or mission-aligned sponsorship) and should not promise permissive licensing or proprietary exclusivity.

## What has concrete implementation evidence in the canonical checkout

The repo is a TypeScript monorepo with React/Vite web client, API, indexer, moderation worker, shared contracts, AT lexicons, and AT OAuth/repository adapters (`README.md:30-47`, `docs/architecture/current-state-matrix.md:34-44`). These are code and documented-test claims, not a fresh validation run in this review.

| Capability | Evidence and maturity stated by the repo | Proposal-safe language |
| --- | --- | --- |
| Public, user-owned AT records | Five Patchwork lexicon families and validation live in `packages/at-lexicons`; `app.patchwork.aid.post` and `app.patchwork.directory.resource` have official-client CRUD routes. The matrix classifies both as `externally-integrated` and describes controlled live-PDS/browser exercises (`current-state-matrix.md:34,36-37`; routes in `services/api/src/index.ts:1289-1448`). | “Built an AT Protocol authoring and projection foundation for public aid and directory records.” |
| OAuth and identity boundary | Official Node OAuth adapter interface is in `packages/at-client/src/oauth-client.ts:1-85`; docs say browser sees cookie/session state rather than OAuth tokens (`current-state-matrix.md:35`). | “Uses an AT OAuth-oriented, session-isolated architecture.” Do not say current production identity operations are proven. |
| Live ingestion and rebuildable projections | Indexer code includes Jetstream v1/v2 sources, PostgreSQL control state, backfill, and projection migration hooks (`services/indexer/src/index.ts:18-26,336-388`; `v2-backfill.ts:71-182`). The matrix labels it externally integrated but explicitly says the v2 cutover lacks fresh protected-staging/live-PDS evidence (`current-state-matrix.md:38`). | “Designed around PDS authority plus rebuildable PostgreSQL projections, with Jetstream-based ingestion and recovery work.” |
| Mutual-aid public discovery | Discovery has durable PostgreSQL read models, public map/feed/directory behavior, block filtering, and strict public-location treatment. Exact resource points are gated by private verification/stewardship/moderation state, while ordinary request geography is approximate (`current-state-matrix.md:39`). | “A privacy-first discovery model for local requests and independently reviewed public resources.” |
| Safety/consent architecture | The accepted ADR prohibits publishing exact personal location, private reports, tokens, blocks, evidence, or internal authorization to AT repositories; it scopes personal precise location to an authorized transient peer channel (`docs/architecture/adr/0003-at-alpha-data-boundaries.md:18-75`). | “Separates portable public records from private safety, moderation, and coordination state by design.” |
| Organization/resource stewardship | Current docs identify durable organization memberships, hashed one-time invitations, stewardship, 90-day reconfirmation, audit trails and notifications as `partially-persistent`; controlled partner operations remain open (`current-state-matrix.md:61`). | “Implemented a private organizational-stewardship model adjacent to public, portable records.” |
| Open civic-resource lineage | The handoff README describes real Chicago-metro resource preview/import and intentionally no demo requests/accounts (`patchwork-mobile-handoff/README.md:27-48`). Its public-resource claims and exact pins carry a claim/review boundary (`README.md:7-10`). | “Has a Chicago public-resource research/import lineage.” Do not claim national completeness, live provider endorsement, or independently verified availability. |

## Safety and protocol principles worth carrying into the umbrella story

1. **User-owned public data, scoped private operations.** The ADR calls the PDS the authority for public aid records and explicitly rejects private operational state as AT data (`adr/0003...:64-119`). This gives the umbrella a credible protocol thesis beyond “add Bluesky login.”
2. **Geoprivacy is implementation-level.** Public requests require coarse/quantized geography; exact public resource locations require ongoing private approval; exact personal coordinates are intentionally non-persistent (`current-state-matrix.md:39`; `adr/0003...:44-62`).
3. **Deletion and reconciliation have a defined authority model.** The ADR makes repository deletion authoritative and requires projection removal/tombstone handling (`adr/0003...:149-177`). This is useful reusable infrastructure, but must be described as Patchwork-specific policy and code, not an AT Protocol standard.
4. **No opaque reputation claim.** The matrix marks reputation as an unwired, fixture-runtime experiment with no production route (`current-state-matrix.md:63`). Do not present scoring or reputation as a deployable platform asset.

## Material maturity and qualification gaps

The README is unusually clear: **“pre-alpha, NO-GO”** (`README.md:9-25`). The matrix defines `production-ready` to require integration, durability, security, deployment, recovery, monitoring and operational ownership, and says no subsystem meets that bar (`current-state-matrix.md:12-22`). Specific conditions that matter to an investor/grant reviewer:

- Fresh credentialed OAuth and managed-signup runs, protected staging repetition, PDS recovery operations, rotation and production operations are incomplete (`current-state-matrix.md:35`).
- Directory partner verification and real partner operations are open (`current-state-matrix.md:37,61`).
- Jetstream v2 needs a full projection comparison, replay, disconnect drill and lifecycle journey before it is operationally verified (`current-state-matrix.md:38`).
- Formal privacy/security/accessibility/translation review, protected alert receipt, and production operations are outstanding across the web/safety surfaces (`current-state-matrix.md:44,51,70`).
- Native mobile and PWA/offline synchronization are only contract-level models; no native app, persistent browser queue, background sync or store pipeline exists (`current-state-matrix.md:65-66`).
- External connectors are fixture-runtime only; there are no partner credentials, durable delivery/outbox, webhooks, or real contract tests (`current-state-matrix.md:68`).
- Operationally, protected promotion, independent durability, a secondary responder and production sizing remain unproven; the README’s historical 40-RPS home-staging result is explicitly not a production-capacity claim (`README.md:12-19`).

## Funding/pitch positioning recommendation

### Strongest role in a Subcult.tv house

Patchwork should be an **open civic-coordination and public-resource interoperability lab** within the AT Protocol house. Its fundable outcome is reusable practice and code for: user-owned public records; permissioned/private safety state; coarse-versus-exact location boundaries; deletion/reconciliation; and organization stewardship. It complements, rather than duplicates, Subcults’ culture/event identity layer and Subcult OS’s operator workflow layer.

### Claims to avoid

- “Patchwork is live/public/production-ready,” “national mutual aid network,” “verified resource directory,” or “proven impact.”
- “AT Protocol solves moderation, safety, consent, or exact location privacy.” Patchwork contributes an application-layer boundary; those problems require governance and operations.
- “Mobile app,” “offline-first,” “integrated 311/crisis services,” or “reputation network.” The canonical matrix labels those as contract-only or fixture-runtime.
- “Current PDS/Jetstream interoperability is broadly proven.” Limited controlled evidence exists; fresh protected-staging and independent compatibility evidence remains open.

### Grant-shaped 12-month work package (not a budget or promise)

1. Re-run reproducible AT CRUD, projection-rebuild, deletion and stream-disconnect exercises with disposable accounts and publish redacted test artifacts.
2. Extract privacy-safe fixtures and conformance tests from the application into reusable AT ecosystem tooling, with clear ownership/governance review.
3. Conduct a narrowly governed Chicago-area protected pilot only after independent privacy, security, accessibility, incident-response and partner-verification gates have evidence.
4. Publish maintainable implementation guidance for public-resource provenance, expiry/reconfirmation and private-versus-public data placement; avoid claiming a universal schema before community review.

## Lightweight adjacent umbrella inventory (README-only, not maturity audits)

These are candidates for a **portfolio map**, not proof that they are all active products or ready to include in an investor deck:

| Project | README-described purpose | Potential umbrella relationship |
| --- | --- | --- |
| `subcult-tv` | Subculture Collective’s curated projects/tools/media/infrastructure presence | Brand and public narrative layer. |
| `subcult-pds` | Invite-only Bluesky PDS for `pds.subcult.tv` | AT identity/infrastructure capability; status must be verified separately. |
| `subcults` | Music-scene mapping, artist/event/touring/live-audio connection around autonomy and privacy | Cultural discovery and public-record flagship. |
| `subcult-os` | Go/React/Expo/Postgres full-stack operating-system project | Private operator/workspace substrate candidate. |
| `patchwork` | AT Protocol-native mutual-aid platform prototype | Civic/public-resource and safety/interoperability R&D vertical. |
| `clpr` | Twitch clip curation/discovery/community/moderation platform | Media/community discovery vertical, not audited here. |
| `hasanara` / `transcript-create` | Citation-first long-form recording/transcript archives | Evidence and research tooling vertical. |
| `soundhash` | Audio matching/fingerprinting across social media | Media provenance/discovery experiment. |
| `assmold` | Searchable Asmongold video/clip archive | Example archive product; audience/domain-specific. |
| `cutroom` | Collaborative short-form video production with AI agents | Production tooling; treat separately from AT thesis. |

## Method and verification boundary

Read: repository-delivery-review rubric; canonical/handoff README and GPL license; state matrix; AT data-boundary ADR; route/indexer/lexicon anchors; worktree/revision/status metadata. The skill’s evidence inventory reported 730 visible files and 178 tests in the canonical checkout, but test file count is not a passing-test claim. I did **not** run `npm` checks, integration/E2E suites, browser journeys, live PDS access, deployments, imports, or any production service action. No secrets, records, or ignored user-owned files were read.

