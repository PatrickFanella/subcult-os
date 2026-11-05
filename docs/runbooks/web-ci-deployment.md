# Web CI deployment

The Gitea workflow runs the existing `make verify` checks and the disposable
PostgreSQL `make test-db` gate before its web image job. Pull requests build and
check the web image. A verified Gitea `main` push then deploys that exact revision
to the existing Kvant `web` service at `https://os.subcult.tv`. Manual dispatch
can retry the current main commit, reusing an immutable image already accepted
by the host.

The production image sets `PUBLIC_WEB_URL=https://os.subcult.tv`, retains nginx
1.29.8 with a pinned registry digest, and checks `/discover` for container health.
Its build context excludes local dependencies, generated output and environment
files. The image smoke check requests discovery and login, verifies the private
page's robots metadata, and requires the relative `/discover` redirect. The
`/favicon.ico` endpoint serves the existing selected `favicon-32.png` bytes;
brand exports and their provenance remain the source for that asset.

`HOMELAB_DEPLOY_KEY` can invoke only the pinned fleet receiver and Subcult OS
profile. `HOMELAB_KNOWN_HOSTS` pins Kvant's SSH host key. The homelab operator owns
the key, rotation and host profile. Deployment steps run only on this Gitea
instance and on `main` push/manual-dispatch events.

The reviewed Compose baseline contains only the existing web service and its
external network, listener and resource limits. The receiver checks current-main
and image identity, retains the previous image and verifies runtime settings,
discovery, redirect, private-page metadata and API health/readiness. Almaz's
existing Umami proxy adds one declared script before `</head>`; the operator's
profile includes that literal when deriving the expected complete discovery HTML.
Other content differences still fail verification. CI compares the live discovery
HTML with that accepted hash and the live favicon with its selected source PNG.

Failed activation or readiness restores the retained image and records private
diagnostics and a rollback command under
`~/.local/state/fleet-deploy/subcult-os/releases/` on Kvant. The frontend is
stateless. Its rollback does not restore or migrate a database.

This lane updates the web image. API/database releases continue to use the
[deployment checklist](deployment-checklist.md) and migration/backup procedures.
Mail, announcement, projection and AT Protocol worker profiles retain their
existing opt-in and qualification requirements. Web deployment does not enable a
worker, change its credentials, run data migrations or send live messages.
Production lifecycle/provider qualification remains separate from CI tests.
