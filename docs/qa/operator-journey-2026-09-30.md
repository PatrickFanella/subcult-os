# Disposable operator journey — 2026-09-30

This is an ongoing synthetic rehearsal, not completion of issue #25. Its first
browser stage found and repaired missing owner role setup. The remaining stages
are staffing/commitments, application versus assignment, participant views,
free booking/ticket/door, finance, private closeout and template reuse. Paired
existing-versus-joined time measurement and intended-user evaluation remain open.

## Runtime identity

- Source baseline: qualified PR #180, `e25312ef23807aef861579ec916bb07965289ad6`.
  Backend source remains that baseline; Vite serves the role-setup candidate from
  the same source-mounted checkout. Published candidate revision belongs to its PR.
- Compose project: `subcult-qa-operator-e25312e`, using only standalone
  `compose.dev.yml` and an ignored disposable-only override.
- Web/API preview: `http://localhost:33027`; PostgreSQL loopback binding33028.
- Database: `subcult_qa_operator_e25312e`, tmpfs storage, all 29 migrations.
  Initial people/events/outbox counts were zero.
- API health and actual database identity were checked. Mail delivery, AT OAuth
  and AT projection are disabled; no mail/announcement/provider worker is started.
- `localhost` cookies/storage are separate from the original retained preview's
  `127.0.0.1` hostname. The retained development database is not a rehearsal target.

Container IDs, configuration, flags and counts are recorded in ignored
`.cache/dev-env/operator-journey-runtime.json`. Preserve this owned disposable
stack while the rehearsal continues, then remove only its resources after receipts
are saved. Do not run mailbox readers or rehearsal scripts against retained data.

## Persisted browser stages

The synthetic owner signed up and explicitly verified through the normal browser
confirmation. The held verification link was read only for its `example.test`
recipient from the disposable DB; no messages were sent. The browser created
workspace “Operator Journey · Disposable” and event “Operator Journey · Free Night”.
The event was then published locally through its normal editor action.

The datetime-locator helper initially typed into the wrong field. Native form
validation blocked that first attempt; no event was created. The title was
corrected and the date entered with the input's native value setter and normal
input/change events before browser submission. The stored start is
`2026-11-08T00:00:00Z`. This does not qualify the native datetime widget or settle
the legacy event-zone model.

Event `523fef0d-42b3-4930-b788-0068ec077889` belongs to workspace
`68d6238f-4c1f-4d6e-b597-95a18db326ab`. It has allocation10, four saved roles
(one public), zero applications and one held verification outbox row. Role setup,
unknown-save recovery and private/public absence proof are detailed in
[participation role setup](../development/participation-role-setup.md).

This stage is real browser-to-API-to-PostgreSQL evidence for signup, workspace,
event and role setup, with explicit synthetic failure/component checks recorded
separately. It is not a full lifecycle, member, provider, mobile or deployment receipt.
