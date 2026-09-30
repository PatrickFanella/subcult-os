# Fresh finance and private closeout API rehearsal

`rtk proxy make finance-closeout-qa` runs a free-only API journey using fresh
synthetic accounts, a workspace and two events on every invocation. It tests
budget/payable/manual-actual separation, append-only corrections and key replay,
Finance grant/removal, free ticket/check-in counts, settlement finalization,
private archive notes, one-time draft reuse, private template application and
CSV/Markdown/printable-HTML exports. It never requests paid checkout.

Use a separately provisioned disposable stack. Set `API_URL` to its loopback HTTP
API and `QA_DATABASE_URL` to the same stack's loopback PostgreSQL database named
`subcult_qa_*`; set `QA_DISPOSABLE_DATABASE=1`. Keep mail delivery disabled and
provider worker profiles stopped. Then run the command above. The harness does
not provision or stop the stack. The retained T3 development database is not a
rehearsal target, and the disposable DB gate from `scripts/dev-env.sh verify`
is a separate check.

The shell wrapper applies `qa-identity.sh` guards before signup. The Python
helper repeats the target guard before loading cookies or making requests.
Verification reads only each synthetic account's held challenge and consumes it
through the normal endpoint. Cookies live in a private temporary directory that
is removed on exit. Host-only localhost cookies are normalized for urllib's
single-label host convention; the localhost origin remains the request boundary.
Capability URLs, tokens, cookies, DTOs and private records are absent from output.

A passing run prints 27 named checks. Failed steps stop the journey with a
bounded error and retain synthetic database records for inspection; the harness
does not reset or delete application data. Each repeat uses fresh actors, so a
previously finalized settlement is not reused. The target can accumulate held
mail and synthetic records; remove only the owned stack when its evidence is
saved and no other rehearsal depends on it.

This proves a normal API workflow on its configured source/runtime. It does not
prove actual browser entry, focus, screen-reader speech, physical camera/QR,
network partition, paid transactions, provider delivery, browser Print/Save as
PDF, accounting-user fit or paired workflow timings. Keep those acceptance gates
separate. The [operator report](operator-journey-2026-09-30.md) records executed
runs and exact revisions.
