# Transactional email

Decision: Resend sends application messages; Proton remains the human inbox and optional reply destination. A sending subdomain such as `notify.subcult.tv` is proposed, not configured. Account creation, sender/domain approval, credentials, DNS and approved-recipient live testing are separate setup gates. Never use account passwords as API credentials or place secrets in issues.

## Provider boundary (#103)

The Go `internal/mail` adapter sends plain-text transactional messages to the fixed HTTPS Resend endpoint. It uses a ten-second timeout, no environment proxy, no redirects, bounded responses and a stable `subcult-email/<outbox UUID>` idempotency key. It returns provider acceptance, not delivery confirmation. No provider response text, subject, recipient or credential is included in returned errors.

Transport failures, ambiguous successful responses, HTTP 408/429/5xx and a concurrent-idempotency 409 are retryable. Changed-payload idempotency conflicts and other rejections are terminal. Retry-After is bounded to one hour. The durable worker must keep retries inside the provider's retention window and never silently reuse an old message identity outside it.

Its tests replace the HTTP transport with synthetic responses; no Resend account or credential was used. Live deliverability remains under #7.

## Durable worker (#104)

Apply migrations before starting a worker. Version 6 adds delivery state without changing or sending historical rows. `held` is the database default. Only new inserts from an API configured with `MAIL_DELIVERY_ENABLED=true` become pending. Sender and reply-to are frozen with each message so later configuration edits cannot change an idempotent retry's payload.

Set `RESEND_API_KEY`, `MAIL_FROM` and optional `MAIL_REPLY_TO` through the deployed secret/configuration store. The owner approved `info@subcult.tv` for replies and the first controlled delivery test; this is not a credential or an assertion of domain verification. `MAIL_FROM` and the sending subdomain still require provider verification. Keep Proton's inbox MX configuration unchanged.

`email-deliver` defaults to aggregate status with no provider requests. `email-deliver -send -limit 10` explicitly processes one batch. For continuous operation, the optional `mail-workers` Compose profile runs `-send -watch`; it still requires mail delivery enabled. New messages created while disabled remain held, even after enabling. There is no bulk-release command for historical rows.

Claims use two-minute leases and acknowledgement fencing. A crash after provider acceptance retries the same outbox UUID/payload, not a new message. Eight attempts, exponential backoff and a bounded Retry-After apply. Message lifetime and first-attempt retry horizon are at most 23 hours, below the documented 24-hour provider window. Identity messages additionally expire with their challenge, and consumed/superseded challenges are not sent. Old, exhausted or ambiguous work is quarantined, never silently marked delivered or automatically replayed with a fresh key. Terminal rows clear message bodies containing one-use links.

`accepted` means Resend accepted the request, not that the recipient received it. Aggregate status exposes counts only. Monitor pending/leased/failed/quarantined/suppressed counts; investigate provider configuration and verify provider-side evidence before manually deciding whether a new message is appropriate. Do not log raw queue bodies, recipients, provider errors or API keys.

## Signed feedback and suppression (#105)

Version 7 adds minimal durable receipts and recipient suppression. Configure `RESEND_WEBHOOK_SECRET` and register `POST /api/resend/webhook` with Resend before enabling delivery. Enabling mail requires a valid signing secret; when unset, the webhook returns 404. Feedback remains active when sending is disabled. Subscribe to `email.delivered`, `email.failed`, `email.suppressed`, `email.bounced` and `email.complained`; other authenticated event types are ignored.

Verification uses the exact raw body, a 64-KiB limit, HMAC-SHA256 with constant-time comparison, and a five-minute timestamp tolerance. Duplicate event IDs are idempotent; reuse with a different message/type returns 409. Store only event ID, provider message ID, event type and processing timestamps—not payload addresses, subjects or error details. Unknown provider IDs remain pending until a matching outbox acknowledgement exists; they never suppress a payload-supplied address.

Feedback rank is monotonic: 0 unknown, 1 delivered to the recipient mail server, 2 failed, 3 suppressed, 4 bounced, 5 complained. Provider acceptance is tracked separately. Delivered events cannot undo adverse outcomes. Only suppressed, bounced or complained events suppress the known outbox recipient, normalized by trimming and lowercasing; generic failures do not. No automatic unsuppression or replay is provided. Already in-flight provider requests cannot be recalled.

Reconciliation processes at most 100 known receipts per call, after each accepted attempt and at batch start. Claims also exclude recipients with unprocessed adverse receipts, so a feedback backlog does not permit a known blocked recipient to be sent. Minimal receipts and suppression entries are retained durably; automatic retention/deletion and an audited operator unsuppression workflow remain deferred. These tables contain private operational data, not public AT records or marketing consent.

Rollback: stop the mail worker and set delivery disabled; retain the additive migration and all delivery identities. Do not delete or reset accepted/ambiguous rows. Stopping sending does not retract messages already accepted by a provider.

## Official contracts checked 2026-09-20; webhook contracts refreshed 2026-09-22

- [Send email](https://resend.com/docs/api-reference/emails/send-email): fixed send endpoint, plain text, reply-to and provider ID.
- [Idempotency keys](https://resend.com/docs/dashboard/emails/idempotency-keys): keys retained for 24 hours; distinguish conflicting content from concurrent requests.
- [Webhook verification](https://resend.com/docs/webhooks/verify-webhooks-requests): verify the raw body and signed headers before interpreting events.
- [Svix signature specification](https://docs.svix.com/receiving/verifying-payloads/how-manual): signature construction and timestamp checks.
- [Event types](https://resend.com/docs/webhooks/event-types) and [bounce payload](https://resend.com/docs/webhooks/emails/bounced): outcome meanings and provider-message correlation.

No transactional message confers marketing consent. Announcement consent/suppression design remains separately scoped.
