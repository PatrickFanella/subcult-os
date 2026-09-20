# Transactional email

Decision: Resend sends application messages; Proton remains the human inbox and optional reply destination. A sending subdomain such as `notify.subcult.tv` is proposed, not configured. Account creation, sender/domain approval, credentials, DNS and approved-recipient live testing are separate setup gates. Never use account passwords as API credentials or place secrets in issues.

## Provider boundary (#103)

The Go `internal/mail` adapter sends plain-text transactional messages to the fixed HTTPS Resend endpoint. It uses a ten-second timeout, no environment proxy, no redirects, bounded responses and a stable `subcult-email/<outbox UUID>` idempotency key. It returns provider acceptance, not delivery confirmation. No provider response text, subject, recipient or credential is included in returned errors.

Transport failures, ambiguous successful responses, HTTP 408/429/5xx and a concurrent-idempotency 409 are retryable. Changed-payload idempotency conflicts and other rejections are terminal. Retry-After is bounded to one hour. The durable worker must keep retries inside the provider's retention window and never silently reuse an old message identity outside it.

This adapter alone does not activate sending. Its tests replace the HTTP transport with synthetic responses; no Resend account or credential was used. Durable queue processing and signed feedback are tracked separately under #104/#105, and live deliverability remains under #7.

## Official contracts checked 2026-09-20

- [Send email](https://resend.com/docs/api-reference/emails/send-email): fixed send endpoint, plain text, reply-to and provider ID.
- [Idempotency keys](https://resend.com/docs/dashboard/emails/idempotency-keys): keys retained for 24 hours; distinguish conflicting content from concurrent requests.
- [Webhook verification](https://resend.com/docs/webhooks/verify-webhooks-requests): verify the raw body and signed headers before interpreting events.
- [Svix signature specification](https://docs.svix.com/receiving/verifying-payloads/how-manual): signature construction and timestamp checks.

No transactional message confers marketing consent. Announcement consent/suppression design remains separately scoped.
