# Indigo OAuth state persistence contribution candidate

Prepared for [Subcult issue #45](https://git.subcult.tv/subculture-collective/subcult-os/issues/45). This package is a tested contribution draft, not an upstream submission or accepted patch. The production dependency and Subcult's fail-closed wrapper remain unchanged.

## Problem

At Indigo commit `41278964ec8e3253e70d4e919dfb8e34211c543d`, `ClientApp.StartAuthFlow` ignores the error returned by `SaveAuthRequestInfo`. After a successful PAR response, it can return an authorization redirect even though the state required for callback processing was never stored. A database outage therefore appears successful until callback recovery fails.

This is an availability and error-propagation defect. This fixture does not demonstrate an authentication bypass or justify such a security claim.

## Evidence

On 2026-09-20 the [GitHub main-commit API](https://api.github.com/repos/bluesky-social/indigo/commits/main) returned the same commit as the application pin. An [exact-symbol issue search](https://github.com/bluesky-social/indigo/issues?q=SaveAuthRequestInfo) returned no matching issues; repeat a broader search before submission because this is not proof that no related discussion exists.

The synthetic test uses an in-memory HTTP transport, metadata and a successful PAR response. It requires no database, DNS, provider account, Subcult server or real identity. It checks that persistence is reached once, that the store error is returned with `errors.Is` support and that no redirect escapes on failure.

- Unmodified pinned source: the test fails with `error=<nil>; want wrapped persistence failure`.
- Apply `fix.patch` to a disposable source copy: the complete `go test -race ./atproto/auth/oauth -count=1` suite passes.
- The read-only dependency checkout under `.blacktower/clonedeps/repos` was not modified.

## Reproduce

Use a disposable checkout of the exact commit. Copy `start_auth_persistence_test.go` into its `atproto/auth/oauth/` directory, then run:

```sh
go test ./atproto/auth/oauth -run TestStartAuthFlowRejectsUnstoredState -count=1
```

Expect failure on the unmodified source. Apply the included patch only in the disposable checkout:

```sh
git apply --check /absolute/path/to/fix.patch
git apply /absolute/path/to/fix.patch
go test -race ./atproto/auth/oauth -count=1
```

Go 1.26.6 was used for the recorded run. The patch adds no dependencies and changes only the ignored-error boundary. A PAR request may remain at the provider until its normal expiry; the application must not present its redirect when local callback state was not persisted.

## Submission draft

Suggested title: `OAuth StartAuthFlow returns a redirect after state persistence fails`.

Include the exact commit, the failing synthetic test, expected wrapped-error/no-redirect behavior, observed nil-error behavior, and the minimal error-check patch. Ask whether maintainers prefer this handling at the helper boundary. Keep the issue focused; do not bundle transport-policy changes.

Indigo's [README contribution guidance](https://github.com/bluesky-social/indigo/blob/41278964ec8e3253e70d4e919dfb8e34211c543d/README.md#contributions) asks contributors to check existing issues and open an issue for discussion before a PR. Its root has no `CONTRIBUTING.md` at this revision; the README is the inspected guidance. Follow current license/contribution terms and record owner approval before submitting. This fixture and patch were prepared with AI assistance and tested as described; retain that disclosure where appropriate.

The current task authorizes Gitea PRs in Subcult OS, not messages or PRs to external maintainers. No upstream issue, PR, email or discussion was posted. After an accepted upstream fix and a separately verified dependency update, reevaluate whether Subcult's persistence-capture wrapper can be removed.

## Reproduction 2026-09-23

Re-ran from a disposable copy under `/tmp/indigo-repro`, outside this repository and outside the read-only `.blacktower/clonedeps` mirror (which was absent in this worktree). Source came from the local module cache (`go env GOMODCACHE`), read-only; nothing under the module cache was modified.

Commands (Go `1.26.6`, `GOPROXY=off`, `GOFLAGS=-mod=mod`, separate `GOCACHE`):

```sh
SRC="$(go env GOMODCACHE)/github.com/bluesky-social/indigo@v0.0.0-20260903211445-41278964ec8e"
cp -r "$SRC" /tmp/indigo-repro/pinned && chmod -R u+w /tmp/indigo-repro/pinned
cp -r "$SRC" /tmp/indigo-repro/patched && chmod -R u+w /tmp/indigo-repro/patched
cp start_auth_persistence_test.go /tmp/indigo-repro/pinned/atproto/auth/oauth/
cp start_auth_persistence_test.go /tmp/indigo-repro/patched/atproto/auth/oauth/

cd /tmp/indigo-repro/pinned
go test ./atproto/auth/oauth -run TestStartAuthFlowRejectsUnstoredState -count=1 -v

cd /tmp/indigo-repro/patched
git apply --check /tmp/indigo-repro/fix.patch   # exit 0
patch -p1 < /tmp/indigo-repro/fix.patch
go test -race ./atproto/auth/oauth -count=1 -v
```

Pinned-source output (unmodified `41278964ec8e3253e70d4e919dfb8e34211c543d`):

```
=== RUN   TestStartAuthFlowRejectsUnstoredState
    start_auth_persistence_test.go:57: StartAuthFlow error=<nil>; want wrapped persistence failure
--- FAIL: TestStartAuthFlowRejectsUnstoredState (0.00s)
FAIL
FAIL	github.com/bluesky-social/indigo/atproto/auth/oauth	0.004s
```

Patched-source output (same commit, `fix.patch` applied, full package suite with `-race`):

```
=== RUN   TestValidateMetadata
--- PASS: TestValidateMetadata (0.00s)
=== RUN   TestValidateMetadataEndpoints
--- PASS: TestValidateMetadataEndpoints (0.00s)
=== RUN   TestResolver
--- PASS: TestResolver (0.00s)
=== RUN   TestStartAuthFlowRejectsUnstoredState
--- PASS: TestStartAuthFlowRejectsUnstoredState (0.00s)
PASS
ok  	github.com/bluesky-social/indigo/atproto/auth/oauth	1.021s
```

Indigo commit: `41278964ec8e3253e70d4e919dfb8e34211c543d` (pseudo-version `v0.0.0-20260903211445-41278964ec8e`, matching `backend/go.mod`). The module cache copy of `oauth.go` at this commit still reads, unchanged:

```go
// persist auth request info
app.Store.SaveAuthRequestInfo(ctx, *info)
```

No changes were needed to the existing test fixture or patch; both applied and ran as documented above without modification.

### Upstream state, checked 2026-09-23

- Searched `github.com/bluesky-social/indigo` issues and PRs (open and closed) for `SaveAuthRequestInfo`: zero results (`https://api.github.com/search/issues?q=repo:bluesky-social/indigo+SaveAuthRequestInfo`).
- Searched for `StartAuthFlow persist`: only an unrelated open PR, `OAuth: create StartAuthFlowWithUserData` (https://github.com/bluesky-social/indigo/pull/1164), which adds a mechanism for stashing caller-defined data in the OAuth `state` string and does not touch `SaveAuthRequestInfo` error handling.
- Searched for `ClientAuthStore`: only a merged, closed PR, `OAuth ClientAuthStore doc tweaks` (https://github.com/bluesky-social/indigo/pull/1159). It fixed a docstring typo and added a duplicate-state guard inside `MemStore.SaveAuthRequestInfo` (returns an error rather than silently overwriting); it does not add or discuss a caller-side check of that error in `StartAuthFlow`.
- Searched for `oauth persist error`: only the already-merged OAuth client SDK PR (https://github.com/bluesky-social/indigo/pull/1100), unrelated to this defect.
- Fetched `https://raw.githubusercontent.com/bluesky-social/indigo/main/atproto/auth/oauth/oauth.go` directly: `main` still calls `app.Store.SaveAuthRequestInfo(ctx, *info)` without checking the returned error, immediately before building the redirect URL. The defect is present on current `main`, not just at the pinned commit.
- The repository has no root `CONTRIBUTING.md` (`https://github.com/bluesky-social/indigo/blob/main/CONTRIBUTING.md` returns 404). The README's "Contributions" section is the operative guidance: open an issue and allow time for discussion before a PR; issues are scoped to bugs/feature requests in the Go atproto implementation; maintainers may not respond and may close without much feedback; avoid large refactors or undiscussed new features. No AI-assistance disclosure policy is stated anywhere in the README or repository root.

This confirms the defect is still open and unaddressed upstream as of this check; nothing was filed or posted.

## Submitter checklist (human, before any upstream submission)

None of the following were performed by this task; they gate an actual submission and must be done by a human maintainer with authorization to publish outside Subcult OS:

- [ ] Confirm authorization to publish this reproduction/patch outside the organization.
- [ ] Create or use a personal GitHub account for the submission (not a shared/service account).
- [ ] Read and comply with Indigo's DCO/CLA requirements if any are introduced before submission (none were found in the README or repo root as of 2026-09-23; recheck at submission time).
- [ ] Open a GitHub issue first, per the README's contribution guidance, and wait for maintainer feedback before opening a PR.
- [ ] Re-run the reproduction against the then-current `main` immediately before submitting, since main was confirmed to still contain the defect on 2026-09-23 but may change.
- [ ] Decide, with maintainers, whether the fix belongs at the `StartAuthFlow` boundary (this patch) or elsewhere (e.g., inside `SaveAuthRequestInfo` implementations), per the open question in the PR draft below.
- [ ] Attach the AI-assistance disclosure below to the issue/PR description.
- [ ] Do not reference internal Subcult issue numbers, credentials, or private infrastructure in the public submission.

## AI-assistance disclosure

Indigo states no AI-assistance policy in its README or repository root as of 2026-09-23; this disclosure is included anyway. This regression test and patch were prepared with AI assistance (an automated coding agent), reviewed for correctness against the pinned source, and verified by running the exact commands and outputs recorded in this document. A human reviewed the diff before it is considered for upstream submission per the checklist above.

## PR description draft

Title: `OAuth StartAuthFlow returns a redirect after state persistence fails`

Body draft:

> **What**: `ClientApp.StartAuthFlow` (`atproto/auth/oauth/oauth.go`) calls `app.Store.SaveAuthRequestInfo(ctx, *info)` and ignores its returned error. If the store fails (e.g., a database outage), `StartAuthFlow` still returns a valid authorization redirect URL, even though the state required to process the callback was never persisted.
>
> **Impact**: A client following the redirect can complete the identity-provider round trip and return with a `code`/`state`, only to have `ProcessCallback` fail because no matching request info was stored. The failure surfaces late, at callback time, instead of immediately at flow start, and is indistinguishable from a successful start until then. This is an availability/error-propagation defect, not an authentication bypass.
>
> **Repro**: attached `start_auth_persistence_test.go` uses an in-memory HTTP transport and a `ClientAuthStore` wrapper whose `SaveAuthRequestInfo` returns a fixed error. On unmodified `main`/this commit, `StartAuthFlow` returns `error=<nil>` and a non-empty redirect. Expected: a wrapped error (`errors.Is` reaches the store error) and an empty redirect.
>
> **Fix**: check the error from `SaveAuthRequestInfo` and return it wrapped, before constructing the redirect URL. See attached `fix.patch` (touches only the one call site, no new dependencies).
>
> **Question for maintainers**: should this check live at the `StartAuthFlow` call site (as patched), inside each `ClientAuthStore` implementation, or both? Happy to adjust based on preference.
>
> **Disclosure**: this reproduction and patch were prepared with AI assistance and reviewed/tested by a human before submission (see repository's stated AI-assistance disclosure, if any, at submission time; none was found as of 2026-09-23).

This draft is not submitted. It is scoped to this one defect only; it intentionally excludes the separate outbound-policy (proxy/redirect) candidate tracked in `docs/development/upstream.md`.
