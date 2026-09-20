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
