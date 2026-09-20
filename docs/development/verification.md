# Verification matrix
Commands below were inspected in repository Makefiles/package scripts on 2026-09-20. They are instructions for future implementation, not claims of execution. Use the appropriate repository root. On this host, prefix shell commands with rtk (examples use rtk proxy).

## OS existing checks
| Command | What it establishes | Prerequisite / limit |
| --- | --- | --- |
| rtk proxy make verify | Repository install, format, lint, contract, unit, build, Compose/template gate | Installs dependencies; fmt can write Go files; excludes live DB/provider/browser qualification |
| rtk proxy make check-contracts | Required property presence/optionality across TS clients | Not serialization, privacy or exact type conformance |
| rtk proxy make test-backend | Non-DB Go suite | Explicitly clears TEST_DATABASE_URL |
| rtk proxy make test-db | Complete app and AT adapter packages with database-backed cases enabled | Disposable TEST_DATABASE_URL required; private fixture schemas; not live-provider/device evidence |
| rtk proxy make test-web | Web Vitest | Installed pinned dependencies |
| rtk proxy make test-mobile | Mobile module tests | Not device behavior |
| rtk proxy make alpha-qa | API free-event lifecycle against running stack | Creates test records; use isolated authorized stack |
| rtk proxy make fake-event-qa | Organizer setup rehearsal | Not real door-pressure proof |
| rtk proxy make alpha-qa-paid | Optional provider-backed paid flow | Test-mode credentials and explicit authorized environment |
| rtk proxy git diff --check | Whitespace/error check | Include untracked docs via dedicated validator as well |

Do not use make reset-db or migrate-reset to fix a test against an existing user database. Those delete local data.

## Subcults source-audit checks
- rtk proxy make verify: Go module integrity only.
- rtk proxy go test ./... -count=1: Go regression suite; libvips/native prerequisites may be required by the existing build setup.
- rtk proxy make test-integration: tagged integration tests; requires the documented Docker/database environment.
- rtk proxy npm --prefix web run test -- --run: full frontend test run through its current test script.
- rtk proxy npm --prefix web run lint and rtk proxy npm --prefix web run build: frontend static/build checks.
- rtk proxy npm run test:e2e: existing E2E package; inspect suite scope before extracting any fixture.
- rtk proxy git diff --check: whitespace check.

Read Subcults docs/TESTING.md and docs/product/PUBLIC_BETA_RELEASE_STATUS.md for prerequisites before running broad tests. These checks help evaluate a candidate invariant; they are not a gate to import the old application and do not prove OS behavior.

## New checks to implement
| ID | Boundary | Smallest disproof |
| --- | --- | --- |
| T-PUBLIC | Go JSON → web/mobile/public AT projection | Add private sentinel to internal DTO; assert absent from public payload including nested values |
| T-MIGRATE | Ordered schema lifecycle | Fresh, replay, concurrency and failed migration preserve ledger/schema invariants |
| T-ACCOUNT | Canonical identity claim | Matching email without current proof cannot merge or claim an account |
| T-SESSION | Session-family rotation | Replay of a rotated token revokes its active descendants |
| T-SYNTAX | Cross-language protocol syntax | Same account/NSID/exact-record fixture differs between pinned Go Indigo and TypeScript `@atproto/syntax`; must fail |
| T-OAUTH-STORE | AT identity-link persistence | Concurrent/replayed/expired state, plaintext secret, extra scope, revoked link or cross-account DID claim is accepted; must fail |
| T-LEX | Cross-language Lexicon conformance | Same valid/invalid corpus differs between Go and `@atproto/lex`; must fail |
| T-LINK | Operator/public occurrence authorization | Different workspace reads/changes the same relation; must deny |
| T-URI | Resolver network boundary | Private-network/unsupported URI input; no outbound private request |
| T-INTENT | Durable write identity | Same key/different payload conflicts; repeated same payload does not duplicate |
| T-CID | Optimistic concurrency | External edit before write; local attempt cannot overwrite it |
| T-UNKNOWN | Remote timeout ambiguity | PDS accepted write but caller timed out; reconcile without new record |
| T-REPLAY | Duplicate/out-of-order ingestion | Older observation cannot replace latest accepted revision |
| T-DELETE | Source deletion | Public projection disappears; private ticket/report stays |
| T-RETAIN | Conditional retained-data migration | If real legacy data is designated, dry-run and migration preserve its approved row/authority invariants; otherwise mark not applicable |
| T-OUTAGE | Local operational isolation | Disable AT connectivity; local free-ticket lifecycle remains functional |
| T-UX | Unified platform journey | Browser/mobile flow exposes account, pending, denied and conflict states |
| T-RESTORE | Recovery | Independent restore reconstructs both mapping and operational state |

These IDs are specifications, not executable files that already exist. Each implementation task must add the corresponding check and update this matrix with its actual command.

## Evidence levels
Source implementation → passing local test → configured integration → real browser/device journey → operational rehearsal → qualified deployed artifact.
Record exact revision, environment, command, timestamp and result at each level. Never promote one level to the next by inference.
