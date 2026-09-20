# DSA Suite: Funding-Pitch Evidence Review

**Review date:** 2026-09-19  
**Scope:** Read-only review of the eight directories under `/home/onnwee/Work/subcult/dsa-suite`. This is repository evidence, not a live deployment audit, legal advice, a representation of any organization, or evidence of DSA endorsement.

## Bottom line

The defensible umbrella story is not “eight deployed DSA products.” It is: **a portfolio of open-source civic and community-infrastructure experiments, with one real AT Protocol foundation (`dsa-proto`), two concrete AT-adjacent consumers (`action-network`, `roberts-rules`), and several private-data / operational patterns that can be generalized beyond political organizations.**

The strongest AT Protocol grant narrative is **public, provenance-bearing organizational records paired with locally governed private operations**. The best general-open-source narrative is reusable authority, audit, consent, moderation, meeting, and evidence-management patterns. Do not use DSA’s name, marks, chapter relationships, member data, campaigns, or electoral outcomes as validation unless a separate written authorization establishes that relationship.

No tests or runtime checks were run in this review. Test surfaces below mean relevant automated tests exist; nothing is labeled test-verified without a current passing run. All runtime and production claims are intentionally excluded unless a README explicitly describes a bounded published release.

## Portfolio map

| Project | Revision and tree | License | Evidence-based role | AT Protocol maturity | Pitch-safe shorthand |
| --- | --- | --- | --- | --- | --- |
| `dsa-proto` | `main` / `cae329388141`, clean | GPL-3.0-only | Public organizational-data AppView foundation | Implemented prototype: temporary Lexicons, Tap sync, OAuth/DPoP publishing, typed client | “Open prototype for public organizational records and authority resolution on AT Protocol.” |
| `action-network` | `main` / `f1c7c265cee3`, clean | GPL-3.0-only | Private CRM, events and messaging with public-record publication boundary | Implemented integration paths, dependent on external foundation/PDS configuration | “Privacy-separated organizing workflow prototype; not an official campaign system.” |
| `dsa-signal` | `main` / `da28d8b024f5`, clean | GPL-3.0-only | Membership-gated Matrix / OIDC control-plane experiment | Indirect: relies on `dsa-proto` identity/authority; no direct AT record system evidenced | “Security-conscious membership communications architecture, locally qualified only.” |
| `roberts-rules` | `main` / `de5ac5a33127`, clean | AGPL-3.0-only | Event-sourced meeting simulator with public-approved artifact boundary | Experimental Lexicons and publication package; explicitly no production PDS integration | “Open meeting-procedure simulator separating public outcomes from private deliberation.” |
| `dsa-forum` | `sync/2026-08-16` / `d8cec7b20cdb`, clean | **No project-wide license found** | Same-origin community forum | Identity handoff dependency only; no direct AT protocol implementation found | “Private-by-design community forum prototype; do not call it open source until licensed.” |
| `dsa-seats` | `codex/nationwide-election-intake-20260821` / `8458a2848680`, **two untracked XLSX files** | **No project-wide license declared** | Citation- and provenance-centered federal-seat research tool | No AT Protocol integration found | “Source-locked civic research architecture; never a voter-profiling or campaign-persuasion product.” |
| `dsa-slate` | `main` / `06d9fc459b5f`, clean | GPL-3.0-only | Generic Go/React/Postgres scaffold | None found | “Reusable scaffold only; not an AT Protocol product or traction evidence.” |
| `dsa-pac` | not a Git checkout; empty directory | none | No implementation | None | “Exclude from pitches and funding inventory.” |

## Project evidence

### dsa-proto — strongest protocol proposal candidate

- **Exact repository evidence:** `main` at `cae329388141`; clean tree; GPL-3.0-only. `README.md` calls it a “TypeScript AppView foundation for public organizational data on AT Protocol.”
- **Implemented source evidence:** Lexicon JSON under `backend/lexicons/us/dsaslate/temp/`; generated bindings under `backend/src/lexicons/`; AT consumption in `backend/src/atproto-live.ts` and `backend/src/ingest-atproto.ts`; authority/data paths in `foundation-service.ts`, `foundation-publish.ts`, `foundation-tap*.ts`, and `foundation-postgres.ts`; OAuth/DPoP implementation and tests in `backend/src/oauth-service.ts` and `.test.ts`; typed client in `packages/foundation-client/`.
- **Supported capability claim:** Temporary Lexicons, deterministic fixture conversion, Tap-backed sync, policy-versioned authority resolution, XRPC/REST view adapters, OAuth/DPoP PDS publishing, encrypted Postgres state, and an isolated events surface are present in code and test paths. The offline gate is defined by `Makefile` as `make verify`; networked PDS/Tap work is explicitly separate as `make verify-integration`.
- **AT maturity:** **Implemented prototype / test-surfaced; not production-verified.** The repository’s own explicit limitation is decisive: `us.dsaslate.temp.*` is provisional, no stable namespace/formally adopted activation exists, and nothing is official DSA infrastructure. The resolver cannot return `official` under current design.
- **Private/public posture:** Public member affiliations are intentionally dropped during migration. OAuth session/state and private review operations remain local. This makes it a credible source of reusable architecture rather than a public directory of real members.
- **Grant-ready work package:** Harden and generalize a protocol-neutral “organization authority and record provenance” kit: compatibility fixtures, temporary-to-stable namespace migration tooling, independent conformance reader, Tap recovery/rebuild tests, and public documentation. Fund only work that does not require private membership records.
- **Caveat:** Do not pitch the `us.dsaslate` namespace, purported organizational authority, DSA chapters, or “official records” as adopted. Say “provisional organizational-data prototype informed by a federated civic use case.”

### action-network — useful privacy and consent companion, high political-separation requirement

- **Exact repository evidence:** `main` at `f1c7c265cee3`; clean tree; GPL-3.0-only.
- **Implemented source evidence:** `backend/src/supporter-*`, `campaign-service.ts`, `event-*.ts`, `workspace-*.ts`, `authority-delegation.ts`, `delegation-oauth.ts`, `delegation-publication-service.ts`, `contact-protection.ts`, `rsvp-abuse.ts`, `registration-delivery.ts`, and their unit/integration tests. Migrations include `0013_privacy_and_webhooks.sql`, `0014_campaign_delivery.sql`, `0015_campaign_provider_events.sql`, and `0018_campaign_delivery_safety.sql`.
- **Supported capability claim:** The README clearly separates signed public organizational records from private supporter, RSVP, consent, messaging, and analytics data. The operations runbook describes supporter export/deletion and queue-scrubbing, which is strong evidence of a deliberate local-data boundary. `Makefile` declares a local `verify` gate and a separate integration test gate requiring `TEST_DATABASE_URL`.
- **AT maturity:** **Implemented application integration, conditional on foundation/PDS configuration; not independently runtime-verified here.** There are publication and OAuth/delegation services, but no basis to imply a deployed PDS integration, partner integration, or real organizer use.
- **Private/public posture:** Supporter information, communications consent, registrations, campaign delivery, webhook payloads, and analytics must remain private. A public record should contain only an explicitly selected, non-sensitive organizational artifact.
- **Grant-ready work package:** Extract the transferable patterns: consent-aware local delivery, public/private publication boundary, delegated organizational authorship, deletion/export contracts, and safe event-registration interfaces. Position a grant as general-purpose civic/community infrastructure, not campaign execution.
- **Caveat:** Its stated audience is DSA chapters and it includes campaign concepts. U.S. charitable funding requires an explicit program firewall and counsel-approved scope. Do not include electoral activity, partisan messaging, coordination, or data from this repository in a charitable grant budget.

### dsa-signal — security architecture, not a safe “encrypted chat product” claim

- **Exact repository evidence:** `main` at `da28d8b024f5`; clean tree; GPL-3.0-only.
- **Implemented source evidence:** Matrix/OIDC composition in `backend/src/composition.ts`, `backend/src/oidc/*`, identity and entitlement modules in `backend/src/identity/*` and `backend/src/entitlements/*`, plus `web/src/auth/matrix-oauth.ts`. Local Matrix/MAS topology is under `deploy/local/`. The repository has extensive test surfaces (72 listed) and contract scripts.
- **Supported capability claim:** A membership-gated Matrix E2EE architecture using `dsa-proto` for identity/organizational authority is implemented to a local-contract level. OIDC state recovery, entitlement/revocation and digest-pinned local topology are concrete code/runbook work.
- **AT maturity:** **Indirect / architectural dependency.** It consumes an external foundation’s identity/authority model; it does not itself evidence AT record publication or a general AT Protocol messaging primitive.
- **Critical limitation:** The README says it is not affiliated with Signal; Matrix E2EE does not protect service-visible metadata; browser clients remain exposed to a compromised origin. Task 8 uses fixture responses and “neither issues tokens nor claims login.” Task 9 blocks without two authorized entitled accounts and external evidence. The root Compose configuration alone does not establish the production control-plane roles.
- **Grant-ready work package:** Open security and interoperability work: membership entitlement/revocation contracts, metadata minimization documentation, OIDC/MAS compatibility fixtures, and reproducible local topology. Treat any end-user secure-communications promise as out of scope until independently audited and live-tested.
- **Caveat:** Do not say “secure Signal alternative,” “production encrypted comms,” or “private by default” without the metadata caveat and new security review. Keep political membership and all real communications out of public AT records.

### roberts-rules — promising public/private meeting architecture, explicitly a simulator

- **Exact repository evidence:** `main` at `de5ac5a33127`; clean tree; AGPL-3.0-only.
- **Implemented source evidence:** Meeting domain commands/projections/replay in `packages/meeting-domain/src/`; publication contract and generated Lexicons in `packages/meeting-publication/`; AT repository adapter in `packages/atproto-publication/src/atproto-repository.ts`; backend event source/store/service files; React live-floor controls in `web/src/`. Tests cover domain, event stream, publication approval/privacy and Playwright paths. `Makefile` defines a broad offline `verify`, separated E2E and packaged/pilot targets.
- **Supported capability claim:** An event-sourced, chair-controlled meeting simulator implements agenda/motion/debate/attendance and several **non-secret simulation-only** vote forms. It has a conscious boundary in which public organization identity, rule profiles, manifests, and explicitly approved aggregate outcomes can be represented through experimental AT schemas, while rosters, credentials, deliberation, attendance, participation records, and ballots stay private.
- **AT maturity:** **Experimental publication adapter and schemas; no production PDS integration.** The README expressly says so.
- **Security/rights limitations:** Secret ballots are prohibited pending an independent security gate; threat models are design work, not a secret-ballot system. The instructions prohibit copying/bundling text from *Robert’s Rules of Order* and prohibit describing simulated voting as binding, secure, or production-ready.
- **Grant-ready work package:** Generalize the open-source, public/private governance record architecture: reproducible event transcripts, approved aggregate-publication contracts, accessible live-floor simulation, and independent privacy/conformance tests. This is credible as democracy-technology R&D, not election administration.
- **Caveat:** Never promise binding governance, secret ballots, ElectionGuard equivalence, legal procedure compliance, or official DSA decision-making.

### dsa-forum — functional local forum with license and deployment gaps

- **Exact repository evidence:** `sync/2026-08-16` at `d8cec7b20cdb`; clean tree. No `AGENTS.md`, CI workflow, or project-wide license file was found.
- **Implemented source evidence:** Fastify app and migrations under `backend/src/app.ts` and `backend/src/db/migrations/001`–`014`; auth, categories, content, follows, lifecycle, notifications, and forum repositories/routes under `backend/src/features/`; community authority/policy modules under `backend/src/community/`; roughly 98 tests listed by evidence collection.
- **Supported capability claim:** Anonymous browsing, identity handoff, topics/replies, immutable revision history, moderation, private notifications/follows/activity and a strong lifecycle/visibility policy matrix have implementation and test surfaces. The README defines local guarded Compose release mechanics, but explicitly says production deployment is unsupported.
- **AT maturity:** **No direct AT Protocol implementation evidenced.** It references a pinned Foundation identity-handoff service and DIDs, but that is an integration boundary, not an AT product claim.
- **Private/public posture:** Authentication handoff, private notification/follow reads, moderator operations, reports, retained content and the local database are sensitive. Public profile links are unverified user-generated destinations and should never be characterized as endorsed.
- **Grant-ready work package:** Only after licensing: package the moderation, reversibility, visibility, cursor, and retention patterns as general community-software work. Fix the missing project license and establish production/release evidence first.
- **Caveat:** Do not call it “open source” in a funding proposal while it has no license. Do not call it deployed or DSA-operated. The member-reporting plan is partially planned material; it should not be presented as finished solely because a migration number appears in a plan.

### dsa-seats — strong provenance research discipline; high partisan/election sensitivity

- **Exact repository evidence:** `codex/nationwide-election-intake-20260821` at `8458a2848680`; two untracked spreadsheets are present: `Congressional Democrat Left Tracker.xlsx` and a Palestine Tracker workbook. No project-wide license is declared.
- **Implemented source evidence:** Next.js/Drizzle/Postgres/PostGIS architecture in `ARCHITECTURE.md`, 18 migrations under `drizzle/`, source locks/acquisition/finalization scripts under `scripts/`, runtime-readiness modules and 400+ test files. Public data and source receipts live under `data/`; NUC operational scripts are under `deploy/nuc/`.
- **Supported capability claim:** Repository documentation supports a reproducible, source-locked federal seat research platform with distinct source/review/publication states, address privacy constraints, migration/role boundaries, tests, and release-evidence structure. The README identifies it as research rather than voter profiling, persuasion, fundraising, or election prediction.
- **Current maturity:** **Implemented research foundation with a limited declared public R1.** `ARCHITECTURE.md` says current published R1 covers 541 offices (441 House, 100 Senate) and 497 valid geometries. It equally says R2–R4 enrichment/maps/corrections/lookup work remains candidate, synthetic, disabled, or blocked. This must be stated alongside the R1 count.
- **AT maturity:** **None found.** It could contribute provenance models or release receipts to an AT-oriented portfolio only as a future, separately scoped integration; do not imply existing AT integration.
- **Private/public posture:** Submitted addresses/coordinates must be transient and excluded from logs, telemetry, storage and public responses. The address endpoint is disabled by default and blocked on edge/egress, vendor, retention, approval and controlled-canary evidence. Source provider reuse conditions govern data redistribution.
- **Grant-ready work package:** Broad, nonpartisan civic-data integrity work: source-lock and correction receipts, explicit missingness, provenance-preserving public release, reproducible geospatial artifacts and privacy-preserving address lookup research. This is more appropriate for civic-tech/open-data funders than an AT Protocol core grant.
- **Caveat:** Despite disciplined scope, it concerns federal elections and candidate/incumbent evidence. Do not pitch it to charitable funders as partisan activity, campaign targeting, electoral strategy, or DSA work. Obtain specialist U.S. nonprofit/election-law advice and place any such work in a separately governed entity, personnel/time ledger, repository/data store, and communications channel. The untracked workbooks mean this checkout should not be treated as a clean distributable artifact without separate provenance review.

### dsa-slate — template, not product evidence

- **Exact repository evidence:** `main` at `06d9fc459b5f`; clean tree; GPL-3.0-only.
- **Implemented source evidence:** Small Go/Vite template: `backend/cmd/app/main.go`, `backend/internal/app/app.go`, `migrations/0001_init.sql`, `web/src/App.tsx`, two basic test files, Docker/Makefile/template docs.
- **Maturity/AT:** **Scaffold/template only; no AT Protocol code or project-specific product implementation found.**
- **Pitch use:** Describe only as a reusable baseline showing codebase hygiene and a minimal full-stack starting point. Exclude it from traction, user, product, interoperability, and funding-request claims.

### dsa-pac — no evidence to present

- **Exact directory evidence:** `/home/onnwee/Work/subcult/dsa-suite/dsa-pac` exists but is empty, has no Git metadata, no README, no license, no code, and no tests.
- **Pitch use:** Exclude entirely. Its name also creates immediate electoral/partisan ambiguity, so it should never be placed in a charitable/open-source funding narrative without a separately defined and counsel-reviewed entity/scope.

## Cross-portfolio funding and affiliation guardrails

1. **No endorsement inference.** “DSA” in a repository name, a technical design reference, a use case, or a user’s own affiliation is not authority to claim DSA sponsorship, customer status, partnership, chapter access, use of marks, access to member records, or approval. Secure written authorization before any such statement.
2. **Separate public protocol R&D from political/electoral work.** A grant proposal can fund protocol interoperability, open Lexicons, migration/test fixtures, accessibility, public documentation, independent security review, generic provenance tooling, and non-electoral community operations. It should not fund campaign activity, candidate support/opposition, voter profiling, persuasion, fundraising, election strategy, member lists, or election administration without specialized legal/compliance review.
3. **Maintain genuine operational separation.** If mixed work continues: use a separate legal entity or documented program where appropriate; dedicated repositories/data stores/cloud accounts; staff time records; separate budgets/bank accounting; access controls; written review gates; no shared contact/membership/campaign datasets; clear external branding. A disclaimer alone is not a firewall.
4. **Do not expose private data to AT Protocol.** For every project, public records need an allowlisted schema and approval step. Keep member status, supporter/RSVP records, consent, addresses, granular location, attendance, communications, roster/credentials, deliberation, ballots, security material and metadata out of public records.
5. **Licensing must be fixed before commercialization or broad grant promises.** GPL projects may be fundable and usable as open source, but the bundle must respect copyleft obligations. `roberts-rules` is AGPL, relevant for network use. `dsa-forum` and `dsa-seats` have no project-wide license; funder-facing open-source claims must exclude them or obtain an explicit license and source-data reuse review.
6. **Use maturity labels in every deck.** Suggested legend: `implemented/test-surfaced`, `local-contract only`, `prototype/provisional`, `planned`, and `not evidenced`. Avoid “production,” “official,” “secure,” “binding,” “deployed,” “adopted,” “interoperable,” or “live” unless a current, independently captured proof supports the precise wording.

## Recommended portfolio segmentation for the parent proposal

| Proposal lane | Include | Exclude or defer | Appropriate deliverable |
| --- | --- | --- | --- |
| AT Protocol core and ecosystem | `dsa-proto`; publication-boundary lessons from `roberts-rules`; delegated-auth/consent patterns from `action-network` | DSA affiliation claims, all real member/supporter data, security/production promises | Open Lexicons, conformance fixtures, safe migration tools, Tap recovery suite, public documentation and independent reader |
| Open community infrastructure | `roberts-rules`; future licensed `dsa-forum`; parts of `dsa-signal` security design | Secret ballots, production E2EE claims, verbatim Robert’s Rules text | Public/private governance-record reference implementation, moderation and reversible-content primitives, accessibility/evaluation artifacts |
| Civic-data integrity | `dsa-seats` provenance and correction architecture | Campaign/race targeting, electoral persuasion, unreviewed workbook content, a current AT integration claim | Source-lock/release-receipt toolkit, explicit-missingness schema, privacy-preserving lookup research, data reuse audit |
| Commercial / service product | `action-network` only after an independent customer and compliance plan | Charitable funds for campaign operations; any implied DSA client status | Locally governed event/CRM workflow sold to non-electoral member organizations, with a clean open-core boundary |

## Verification record and limitations

- Applied the repository-delivery-review evidence rubric and ran `collect_repository_evidence.py` against each Git repository. This yielded file/test inventories and revision/tree state.
- Read applicable project `AGENTS.md` files before inspection for `dsa-proto`, `dsa-slate`, `action-network`, `dsa-signal`, `roberts-rules`, and `dsa-seats`. `dsa-forum` has no `AGENTS.md`; `dsa-pac` is empty.
- Read core README/architecture/runbook and source-path evidence; performed no build, test, PDS, database, browser, credentialed, production, outreach, donor, or deployment actions.
- This does not validate source-data licenses, trademark permission, nonprofit tax status, campaign-finance compliance, security properties, data-protection law, user demand, or any funder’s current eligibility criteria. Those require separate primary-source research and qualified legal/compliance review.

