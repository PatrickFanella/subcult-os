# External ticket purchase-link handoffs

Status: private operator configuration and preview only (TICKET-EXT, migration 000022).

An owner or organizer may configure one HTTPS purchase URL and the provider's
honest display label for a canonical `event_occurrences` row. The target
hostname must exactly match `EXTERNAL_TICKET_ALLOWED_HOSTS`, a comma-separated
runtime allowlist. The default is empty and denies every URL. A configured
link is not an approved provider integration or a claim that the provider has
been contacted.

`GET` and `PUT /api/events/{eventID}/occurrences/{occurrenceID}/external-ticket-handoff`
are authenticated operator routes. `PUT` requires `expectedUpdatedAt`: use an
empty value for the first configuration, then send the returned revision for
every replacement. A stale revision receives `409`, preserving a concurrent
operator's update. Revoked and expired memberships fail the same active-member
check as other operator routes.

This slice has no anonymous renderer. The table is private and public
discovery continues to read only independently projected `at_projection_records`.
No configuration here publishes an occurrence, creates tickets or checkout
sessions, fetches a URL, imports provider data, calls a payment API, or accepts
webhooks. A future public renderer must independently prove that its selected
occurrence is published and allowed before exposing this data; it may not use
the handoff table to find private events.

The permitted test fixture hostname is `tickets.example.test`. It is not a
provider choice or production configuration.
