# Stripe Paid Ticketing Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Add a first paid-ticket path using Stripe Checkout while preserving the existing free-ticket lifecycle.

**Architecture:** Keep `subcult-os` as a monolithic Go API with PostgreSQL-backed state and a React public event page. Organizers configure one Event-level pricing mode; guests reserve free tickets immediately or start a Stripe Checkout Session for paid tickets; tickets are fulfilled only from the signed Stripe webhook, never from the browser redirect.

**Tech Stack:** Go 1.26, PostgreSQL 17, pgx/pgxpool, `github.com/stripe/stripe-go/v85`, Stripe Checkout Sessions, Stripe webhooks, Vite React TypeScript, Docker Compose, Make.

---

## Source Guidance

- ADR: `docs/adr/0002-stripe-payment-provider.md` says Stripe is assumed for ticket checkout, while domain terms remain provider-neutral.
- Official Stripe docs checked for this plan:
  - Checkout quickstart: `https://docs.stripe.com/checkout/quickstart`
  - Create Checkout Session: `https://docs.stripe.com/api/checkout/sessions/create`
  - Fulfillment: `https://docs.stripe.com/checkout/fulfillment`
  - Webhooks: `https://docs.stripe.com/webhooks`
  - Signature verification: `https://docs.stripe.com/webhooks/signature`
  - Idempotency: `https://docs.stripe.com/api/idempotent_requests`
  - Local Stripe CLI: `https://docs.stripe.com/get-started/development-environment`

---

## File Structure

- Modify `backend/internal/app/config.go`: add Stripe secret key, webhook secret, and paid-ticket feature configuration.
- Modify `backend/internal/app/schema.sql`: add price/payment/order fields and webhook-event idempotency table.
- Create `backend/internal/app/payments.go`: isolate Stripe Checkout Session creation and webhook handling.
- Modify `backend/internal/app/app.go`: register payment and webhook routes.
- Modify `backend/internal/app/events.go`: include pricing fields in Event DTOs, create/update validation, public event response, report counts.
- Modify `backend/internal/app/tickets.go`: keep free reservation path; add paid reservation intent path; do not create paid tickets until webhook success.
- Modify `backend/internal/app/lifecycle_test.go`: add DB-backed paid-ticket tests using a fake payment provider seam.
- Modify `web/src/domain.ts`: add pricing/payment DTO fields.
- Modify `web/src/views/EventEditorView.tsx`: allow Owners/Members to configure free vs fixed paid ticket pricing before publication.
- Modify `web/src/views/PublicEventView.tsx`: show free vs paid CTA and redirect paid reservations to Checkout URL.
- Modify `scripts/alpha-qa.sh`: keep default free-ticket QA; add documented optional paid mode later when Stripe CLI env is present.
- Modify `.env.example`, `docker-compose.yml`, and `README.md`: document Stripe local environment and webhook testing.

---

## Task 1: Payment Domain Schema and Config

**Files:**
- Modify: `backend/internal/app/config.go`
- Modify: `backend/internal/app/schema.sql`
- Modify: `backend/internal/app/app_test.go`
- Modify: `.env.example`
- Modify: `README.md`

- [ ] **Step 1: Add config tests**

Add tests to `backend/internal/app/app_test.go`:

```go
func TestConfigValidateAllowsPaidTicketingDisabled(t *testing.T) {
	config := Config{AppEnv: "production", Addr: ":8080", DatabaseURL: "postgres://app:secret@db:5432/app?sslmode=require", SessionSecret: "replace-with-a-long-random-secret", PublicWebURL: "https://subcult.example"}
	if err := config.Validate(); err != nil {
		t.Fatalf("production config without Stripe should validate when paid ticketing is disabled: %v", err)
	}
}

func TestConfigValidateRejectsPartialStripeConfig(t *testing.T) {
	config := Config{AppEnv: "production", Addr: ":8080", DatabaseURL: "postgres://app:secret@db:5432/app?sslmode=require", SessionSecret: "replace-with-a-long-random-secret", PublicWebURL: "https://subcult.example", StripeSecretKey: "sk_test_123"}
	err := config.Validate()
	if err == nil || !strings.Contains(err.Error(), "STRIPE_WEBHOOK_SECRET") {
		t.Fatalf("expected webhook secret validation error, got %v", err)
	}
}

func TestConfigValidateAllowsCompleteStripeConfig(t *testing.T) {
	config := Config{AppEnv: "production", Addr: ":8080", DatabaseURL: "postgres://app:secret@db:5432/app?sslmode=require", SessionSecret: "replace-with-a-long-random-secret", PublicWebURL: "https://subcult.example", StripeSecretKey: "sk_test_123", StripeWebhookSecret: "whsec_123"}
	if err := config.Validate(); err != nil {
		t.Fatalf("complete Stripe config should validate: %v", err)
	}
}
```

- [ ] **Step 2: Add config fields**

In `backend/internal/app/config.go`, extend `Config`:

```go
StripeSecretKey     string
StripeWebhookSecret string
```

Load them in `LoadConfig()`:

```go
StripeSecretKey:     env("STRIPE_SECRET_KEY", ""),
StripeWebhookSecret: env("STRIPE_WEBHOOK_SECRET", ""),
```

Update `Validate()` so if exactly one of the Stripe values is set in production it rejects with the missing variable name. If both are empty, paid ticketing is disabled and production config remains valid.

- [ ] **Step 3: Add schema columns/tables**

Append idempotent schema changes to `backend/internal/app/schema.sql`:

```sql
alter table events add column if not exists pricing_mode text not null default 'free' check (pricing_mode in ('free', 'fixed'));
alter table events add column if not exists ticket_price_cents integer not null default 0 check (ticket_price_cents >= 0);
alter table events add column if not exists ticket_currency text not null default 'usd';

alter table tickets add column if not exists payment_status text not null default 'free' check (payment_status in ('free', 'pending', 'paid', 'cancelled'));
alter table tickets add column if not exists amount_cents integer not null default 0 check (amount_cents >= 0);
alter table tickets add column if not exists currency text not null default 'usd';
alter table tickets add column if not exists stripe_checkout_session_id text unique;
alter table tickets add column if not exists paid_at timestamptz;

create table if not exists payment_webhook_events (
  id text primary key,
  provider text not null default 'stripe',
  event_type text not null,
  processed_at timestamptz not null default now()
);
```

- [ ] **Step 4: Document environment**

In `.env.example`, add commented local values:

```dotenv
# STRIPE_SECRET_KEY=sk_test_replace-me
# STRIPE_WEBHOOK_SECRET=whsec_replace-me
```

In `README.md`, add a short “Paid ticketing local setup” note saying Stripe keys are optional until a paid Event is configured and webhooks are tested with `stripe listen --forward-to localhost:38080/api/stripe/webhook`.

- [ ] **Step 5: Verify and commit**

Run:

```bash
make verify
```

Expected: all checks pass.

Commit:

```bash
git add backend/internal/app/config.go backend/internal/app/schema.sql backend/internal/app/app_test.go .env.example README.md
git commit -m "Add paid ticketing schema and config"
```

---

## Task 2: Event Pricing API

**Files:**
- Modify: `backend/internal/app/events.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `web/src/domain.ts`

- [ ] **Step 1: Add event pricing fields**

Extend `eventDTO`, `eventRow`, `createEventRequest`, and `updateEventRequest` with:

```go
PricingMode      string  `json:"pricingMode"`
TicketPriceCents int     `json:"ticketPriceCents"`
TicketCurrency   string  `json:"ticketCurrency"`
```

Use pointers for update request fields. Defaults: `pricingMode="free"`, `ticketPriceCents=0`, `ticketCurrency="usd"`.

- [ ] **Step 2: Validate pricing**

Add helper behavior in `events.go`:

- `free`: price must be `0`, currency normalized to `usd`.
- `fixed`: price must be at least `50` cents, currency must be `usd` for first slice.
- Pricing cannot change after any tickets exist.
- Existing published edit constraints still apply.

- [ ] **Step 3: Update event queries**

Update all event `select`/`insert`/`update`/DTO scan paths to include pricing fields. Public events must expose pricing so the guest form knows whether to reserve immediately or start paid checkout.

- [ ] **Step 4: Add backend tests**

In `backend/internal/app/lifecycle_test.go`, add DB-backed tests:

- creating a fixed-price event returns `pricingMode="fixed"`, `ticketPriceCents=1500`, `ticketCurrency="usd"`.
- fixed pricing below 50 cents returns 400.
- changing pricing after a free reserved ticket returns 409.

- [ ] **Step 5: Update frontend types**

In `web/src/domain.ts`, add to `EventDTO`:

```ts
pricingMode: 'free' | 'fixed';
ticketPriceCents: number;
ticketCurrency: string;
```

- [ ] **Step 6: Verify and commit**

Run:

```bash
make verify
```

Commit:

```bash
git add backend/internal/app/events.go backend/internal/app/lifecycle_test.go web/src/domain.ts
git commit -m "Add event pricing API"
```

---

## Task 3: Stripe Checkout Session Creation

**Files:**
- Create: `backend/internal/app/payments.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/tickets.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`

- [ ] **Step 1: Add Stripe SDK**

Run from `backend/`:

```bash
go get github.com/stripe/stripe-go/v85
go mod tidy
```

- [ ] **Step 2: Add payment provider seam**

Create `backend/internal/app/payments.go` with a small interface:

```go
type checkoutSessionRequest struct {
	TicketID    string
	EventID     string
	EventTitle  string
	AmountCents int
	Currency    string
	SuccessURL  string
	CancelURL   string
}

type checkoutSessionResponse struct {
	ID  string
	URL string
}

type paymentProvider interface {
	CreateCheckoutSession(ctx context.Context, req checkoutSessionRequest) (checkoutSessionResponse, error)
}
```

Add Stripe implementation using `stripe.NewClient(config.StripeSecretKey)`, `sc.V1CheckoutSessions.Create(ctx, params)`, `Mode=payment`, `PaymentMethodTypes=[card]`, `PriceData` from trusted server-side Event price, `ClientReferenceID=ticketID`, metadata `ticket_id` and `event_id`, and `params.SetIdempotencyKey("checkout-ticket-"+ticketID)`.

- [ ] **Step 3: Wire provider into App**

Add `payments paymentProvider` to `App`. In `New`, if `StripeSecretKey` is non-empty, initialize the Stripe provider. Tests can set `app.payments = fakeProvider` after `New(...)`.

- [ ] **Step 4: Add paid reservation endpoint**

Register:

```go
a.mux.HandleFunc("POST /api/public/events/{slug}/paid-reservations", a.handleCreatePaidReservation)
```

In `tickets.go`, implement `handleCreatePaidReservation`:

- require DB.
- decode same email/display name request.
- load published event with row lock.
- require `pricing_mode='fixed'`.
- enforce capacity against all non-cancelled tickets.
- create a ticket with `payment_status='pending'`, `status='reserved'`, trusted amount/currency, and a code.
- call payment provider with success URL `/tickets/{code}?checkout=success` and cancel URL `/e/{slug}?checkout=cancelled`.
- save `stripe_checkout_session_id` on the ticket.
- audit `ticket.payment_started`.
- return `{ticketId, checkoutSessionId, checkoutUrl}`.

- [ ] **Step 5: Add tests with fake provider**

Add lifecycle tests:

- paid reservation without Stripe provider returns 503.
- paid reservation for fixed event returns checkout URL and creates pending ticket.
- free reservation endpoint still works for free event.

- [ ] **Step 6: Verify and commit**

Run:

```bash
cd backend && go test ./internal/app
make verify
```

Commit:

```bash
git add backend/go.mod backend/go.sum backend/internal/app/app.go backend/internal/app/payments.go backend/internal/app/tickets.go backend/internal/app/lifecycle_test.go
git commit -m "Start paid ticket checkout sessions"
```

---

## Task 4: Stripe Webhook Fulfillment

**Files:**
- Modify: `backend/internal/app/payments.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/lifecycle_test.go`

- [ ] **Step 1: Register webhook route**

Register:

```go
a.mux.HandleFunc("POST /api/stripe/webhook", a.handleStripeWebhook)
```

Webhook requests must not require cookie auth or origin checks. They must verify the Stripe signature from the raw request body.

- [ ] **Step 2: Implement signature verification and idempotency**

Use:

```go
payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), a.config.StripeWebhookSecret)
```

Before processing, insert `event.ID` into `payment_webhook_events`; if duplicate, return 200. Handle only `checkout.session.completed`; return 200 for ignored event types.

- [ ] **Step 3: Fulfill tickets from completed Checkout Sessions**

For `checkout.session.completed`:

- unmarshal `event.Data.Raw` into `stripe.CheckoutSession`.
- read `ticket_id` metadata and `session.ID`.
- lock matching ticket where `stripe_checkout_session_id=session.ID`.
- set `payment_status='paid'`, `paid_at=now()` only if currently `pending`.
- enqueue ticket email.
- audit `ticket.payment_completed`.

Return 200 after successful processing.

- [ ] **Step 4: Add tests**

Add DB-backed tests for the provider-independent fulfillment helper:

- completing a pending ticket marks it paid and enqueues email.
- running the same webhook event twice is idempotent.
- wrong session ID does not mark unrelated ticket paid.

Signature-level tests can use Stripe’s webhook helper if practical; otherwise cover fulfillment helper and document Stripe CLI manual verification.

- [ ] **Step 5: Verify and commit**

Run:

```bash
make verify
```

Commit:

```bash
git add backend/internal/app/payments.go backend/internal/app/app.go backend/internal/app/lifecycle_test.go
git commit -m "Fulfill paid tickets from Stripe webhooks"
```

---

## Task 5: Paid Ticket Frontend Flow

**Files:**
- Modify: `web/src/domain.ts`
- Modify: `web/src/views/EventEditorView.tsx`
- Modify: `web/src/views/PublicEventView.tsx`
- Modify: `web/src/views/TicketView.tsx`
- Modify: `web/src/App.test.tsx`

- [ ] **Step 1: Add frontend DTOs**

Add:

```ts
export interface PaidReservationDTO {
  ticketId: string;
  checkoutSessionId: string;
  checkoutUrl: string;
}
```

Extend `TicketDTO` with:

```ts
paymentStatus: 'free' | 'pending' | 'paid' | 'cancelled';
amountCents: number;
currency: string;
```

- [ ] **Step 2: Event editor pricing controls**

Add a pricing panel before publication:

- Free reservation.
- Fixed paid ticket in USD.
- Price input in dollars converted to cents.
- Disable pricing edits once tickets exist or Event is End of Night.

- [ ] **Step 3: Public Event paid CTA**

In `PublicEventView`, if `pricingMode === 'fixed'`, button text is `Buy ticket`, POST to `/api/public/events/{slug}/paid-reservations`, and redirect `window.location.href = checkoutUrl`. Free events keep existing in-place confirmation.

- [ ] **Step 4: Ticket page payment status**

In `TicketView`, show a clear paid/free/pending banner. Pending paid tickets should say checkout may still be processing and the ticket is not final until payment completes.

- [ ] **Step 5: Tests**

Update route smoke tests for:

- event editor pricing copy.
- public paid ticket CTA static copy.
- ticket payment status copy.

- [ ] **Step 6: Verify and commit**

Run:

```bash
pnpm --dir web run test
make verify
```

Commit:

```bash
git add web/src/domain.ts web/src/views/EventEditorView.tsx web/src/views/PublicEventView.tsx web/src/views/TicketView.tsx web/src/App.test.tsx
git commit -m "Add paid ticket frontend flow"
```

---

## Task 6: QA, Docs, and Stripe Local Runbook

**Files:**
- Modify: `README.md`
- Create: `docs/runbooks/stripe-local-testing.md`
- Modify: `scripts/alpha-qa.sh`
- Modify: `Makefile`

- [ ] **Step 1: Document Stripe local testing**

Create `docs/runbooks/stripe-local-testing.md` with:

- set `STRIPE_SECRET_KEY`.
- run `stripe listen --forward-to localhost:38080/api/stripe/webhook`.
- copy `whsec_...` into `STRIPE_WEBHOOK_SECRET`.
- use Stripe test card `4242 4242 4242 4242`.
- never fulfill from success redirect; webhook is source of truth.

- [ ] **Step 2: Add optional paid QA command**

Add `make alpha-qa-paid` that runs `scripts/alpha-qa.sh --paid` only when Stripe env is present. The script should skip with a clear message if keys are missing.

- [ ] **Step 3: Update README**

Link the Stripe runbook and clarify free QA vs paid QA.

- [ ] **Step 4: Verify and commit**

Run:

```bash
make verify
make alpha-qa
```

If Stripe CLI/env is available, also run:

```bash
make alpha-qa-paid
```

Commit:

```bash
git add README.md docs/runbooks/stripe-local-testing.md scripts/alpha-qa.sh Makefile
git commit -m "Document Stripe paid ticket QA"
```

---

## Self-Review

- Spec coverage: plan covers backend config/schema, pricing API, Checkout Session creation, webhook fulfillment, frontend paid flow, and local Stripe QA docs.
- Safety: fulfillment happens only from verified webhook, not browser redirect; webhook idempotency table prevents duplicate processing.
- Scope cut: no Stripe Connect payouts, refunds, multiple ticket types, taxes, discounts, coupons, Apple Pay tuning, formal order management, or resale.
- Migration risk: schema uses idempotent `alter table ... add column if not exists` consistent with current alpha migration runbook.
