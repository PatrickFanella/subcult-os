package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v85"
	stripewebhook "github.com/stripe/stripe-go/v85/webhook"
)

type failingCheckoutProvider struct{}

func (failingCheckoutProvider) CreateCheckoutSession(context.Context, checkoutSessionRequest) (checkoutSessionResponse, error) {
	return checkoutSessionResponse{}, errors.New("provider transport timeout: internal detail must not escape")
}

func TestStripeWebhookHTTPRejectsTamperingAndAcceptsSignedCallback(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 1, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	ticketID, sessionID := insertPendingStripeTicket(t, fx, eventID, "signed@example.test", "Signed", testStripeSessionID(t, "cs_signed"))
	fx.app.config.StripeWebhookSecret = "whsec_test_callback"
	raw, err := json.Marshal(map[string]any{"id": testStripeEventID(t, "evt_signed"), "api_version": stripe.APIVersion, "object": "event", "type": "checkout.session.completed", "data": map[string]any{"object": map[string]any{"id": sessionID, "object": "checkout.session", "payment_status": "paid", "amount_total": 1500, "currency": "usd", "metadata": map[string]string{"ticket_id": ticketID, "event_id": eventID}}}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	signature := fmt.Sprintf("t=%d,v1=%x", now.Unix(), stripewebhook.ComputeSignature(now, raw, fx.app.config.StripeWebhookSecret))
	if _, err := stripewebhook.ConstructEvent(raw, signature, fx.app.config.StripeWebhookSecret); err != nil {
		t.Fatalf("invalid synthetic event envelope: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", bytes.NewReader(raw))
	request.Header.Set("Stripe-Signature", signature)
	response := httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("signed webhook status=%d body=%s", response.Code, response.Body.String())
	}
	assertPaidTicketState(t, fx, ticketID, sessionID)
	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)-1] ^= 1
	request = httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", bytes.NewReader(tampered))
	request.Header.Set("Stripe-Signature", signature)
	response = httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("tampered webhook status=%d body=%s", response.Code, response.Body.String())
	}
}

type callbackDuringCreateProvider struct {
	app *App
	t   *testing.T
}

func (p callbackDuringCreateProvider) CreateCheckoutSession(ctx context.Context, req checkoutSessionRequest) (checkoutSessionResponse, error) {
	p.t.Helper()
	var status string
	if err := p.app.db.QueryRow(ctx, `select status from payment_checkout_attempts where id = $1 and ticket_id = $2`, req.CheckoutAttemptID, req.TicketID).Scan(&status); err != nil || status != "creating" {
		p.t.Fatalf("provider was called before durable attempt commit: status=%q err=%v", status, err)
	}
	sessionID := testStripeSessionID(p.t, "cs_during_create")
	if err := p.app.processStripeWebhookEvent(ctx, checkoutAttemptEvent(p.t, testStripeEventID(p.t, "evt_during_create"), sessionID, req.TicketID, req.EventID, req.CheckoutAttemptID)); err != nil {
		p.t.Fatalf("process callback during provider call: %v", err)
	}
	return checkoutSessionResponse{ID: sessionID, URL: "https://checkout.example/during-create"}, nil
}

func checkoutAttemptEvent(t *testing.T, eventID, sessionID, ticketID, eventRefID, attemptID string) stripe.Event {
	return checkoutAttemptEventType(t, stripe.EventTypeCheckoutSessionCompleted, eventID, sessionID, ticketID, eventRefID, attemptID, 1500, "usd")
}

func checkoutAttemptEventType(t *testing.T, eventType stripe.EventType, eventID, sessionID, ticketID, eventRefID, attemptID string, amount int64, currency string) stripe.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"id": sessionID, "object": "checkout.session", "payment_status": "paid", "amount_total": amount, "currency": currency, "metadata": map[string]string{"ticket_id": ticketID, "event_id": eventRefID, "checkout_attempt_id": attemptID}})
	if err != nil {
		t.Fatal(err)
	}
	return stripe.Event{ID: eventID, Type: eventType, Data: &stripe.EventData{Raw: raw}}
}

func TestExpiredWebhookReconcilesUnknownAttemptAndCompletedAfterExpiryIsIgnored(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 1, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	fx.app.payments = failingCheckoutProvider{}
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("expired"), "displayName": "Expired"}, http.StatusBadGateway)
	var ticketID, attemptID string
	if err := fx.app.db.QueryRow(t.Context(), `select t.id, a.id from tickets t join payment_checkout_attempts a on a.ticket_id=t.id where t.event_id=$1`, eventID).Scan(&ticketID, &attemptID); err != nil {
		t.Fatal(err)
	}
	sessionID := testStripeSessionID(t, "cs_expired")
	if err := fx.app.processStripeWebhookEvent(t.Context(), checkoutAttemptEventType(t, stripe.EventTypeCheckoutSessionExpired, testStripeEventID(t, "evt_expired"), sessionID, ticketID, eventID, attemptID, 1500, "usd")); err != nil {
		t.Fatal(err)
	}
	var ticketStatus, attemptStatus string
	if err := fx.app.db.QueryRow(t.Context(), `select t.payment_status, a.status from tickets t join payment_checkout_attempts a on a.ticket_id=t.id where t.id=$1`, ticketID).Scan(&ticketStatus, &attemptStatus); err != nil {
		t.Fatal(err)
	}
	if ticketStatus != "cancelled" || attemptStatus != "expired" {
		t.Fatalf("expiry state ticket=%s attempt=%s", ticketStatus, attemptStatus)
	}
	lateEventID := testStripeEventID(t, "evt_after_expiry")
	if err := fx.app.processStripeWebhookEvent(t.Context(), checkoutAttemptEvent(t, lateEventID, sessionID, ticketID, eventID, attemptID)); err != nil {
		t.Fatal(err)
	}
	assertEmailOutboxCount(t, fx, ticketID, 0)
	var linkedTicketID, linkedAttemptID string
	if err := fx.app.db.QueryRow(t.Context(), `select ticket_id::text, checkout_attempt_id::text from payment_webhook_events where id = $1`, lateEventID).Scan(&linkedTicketID, &linkedAttemptID); err != nil {
		t.Fatal(err)
	}
	if linkedTicketID != ticketID || linkedAttemptID != attemptID {
		t.Fatalf("late callback provenance ticket=%q attempt=%q", linkedTicketID, linkedAttemptID)
	}
}

func TestMismatchedCheckoutTupleCannotBindUnknownAttempt(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 1, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	fx.app.payments = failingCheckoutProvider{}
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("mismatch"), "displayName": "Mismatch"}, http.StatusBadGateway)
	var ticketID, attemptID string
	if err := fx.app.db.QueryRow(t.Context(), `select t.id, a.id from tickets t join payment_checkout_attempts a on a.ticket_id=t.id where t.event_id=$1`, eventID).Scan(&ticketID, &attemptID); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.processStripeWebhookEvent(t.Context(), checkoutAttemptEventType(t, stripe.EventTypeCheckoutSessionCompleted, testStripeEventID(t, "evt_bad_amount"), testStripeSessionID(t, "cs_bad_amount"), ticketID, eventID, attemptID, 1499, "usd")); err != nil {
		t.Fatal(err)
	}
	var session string
	if err := fx.app.db.QueryRow(t.Context(), `select coalesce(provider_session_id, '') from payment_checkout_attempts where id=$1`, attemptID).Scan(&session); err != nil {
		t.Fatal(err)
	}
	if session != "" {
		t.Fatalf("mismatched callback bound session %q", session)
	}
}

func TestPaidReservationPersistsAttemptBeforeProviderAndBindsStableKey(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	fake := &fakePaymentProvider{response: checkoutSessionResponse{ID: testStripeSessionID(t, "cs_attempt"), URL: "https://checkout.example/attempt"}}
	fx.app.payments = fake
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("attempt"), "displayName": "Attempt"}, http.StatusOK)

	if fake.request.CheckoutAttemptID == "" || !strings.HasPrefix(fake.request.ProviderIdempotencyKey, "checkout-attempt-") {
		t.Fatalf("provider request lacks durable attempt identity: %#v", fake.request)
	}
	var status, storedKey, storedSession string
	if err := fx.app.db.QueryRow(t.Context(), `
		select a.status, a.provider_idempotency_key, coalesce(a.provider_session_id, '')
		from payment_checkout_attempts a join tickets t on t.id = a.ticket_id
		where t.id = $1
	`, fake.request.TicketID).Scan(&status, &storedKey, &storedSession); err != nil {
		t.Fatal(err)
	}
	if status != "ready" || storedKey != fake.request.ProviderIdempotencyKey || storedSession != fake.response.ID {
		t.Fatalf("unexpected durable checkout attempt: status=%q key=%q session=%q", status, storedKey, storedSession)
	}
}

func TestPaidReservationReturnsPaidReceiptWhenCallbackArrivesDuringProviderCreate(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 1, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	fx.app.payments = callbackDuringCreateProvider{app: fx.app, t: t}
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	resp := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("during"), "displayName": "During"}, http.StatusOK)
	ticketID := mustString(t, resp.JSON, "ticketId")
	assertPaidTicketState(t, fx, ticketID, mustString(t, resp.JSON, "checkoutSessionId"))
	assertEmailOutboxCount(t, fx, ticketID, 1)
}

func TestCompletedWebhookReconcilesUnknownAttemptAndRejectsDuplicate(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 1, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	fx.app.payments = failingCheckoutProvider{}
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("reconcile"), "displayName": "Reconcile"}, http.StatusBadGateway)
	var ticketID, attemptID string
	if err := fx.app.db.QueryRow(t.Context(), `select t.id, a.id from tickets t join payment_checkout_attempts a on a.ticket_id = t.id where t.event_id = $1`, eventID).Scan(&ticketID, &attemptID); err != nil {
		t.Fatal(err)
	}
	sessionID := testStripeSessionID(t, "cs_reconciled")
	first := checkoutAttemptEvent(t, testStripeEventID(t, "evt_reconciled"), sessionID, ticketID, eventID, attemptID)
	if err := fx.app.processStripeWebhookEvent(t.Context(), first); err != nil {
		t.Fatal(err)
	}
	assertPaidTicketState(t, fx, ticketID, sessionID)
	assertEmailOutboxCount(t, fx, ticketID, 1)
	duplicate := checkoutAttemptEvent(t, testStripeEventID(t, "evt_reconciled_duplicate"), sessionID, ticketID, eventID, attemptID)
	if err := fx.app.processStripeWebhookEvent(t.Context(), duplicate); err != nil {
		t.Fatal(err)
	}
	assertEmailOutboxCount(t, fx, ticketID, 1)
}

func TestPaidReservationRetainsProviderSuccessWhenSessionBindingConflicts(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	conflictingSession := testStripeSessionID(t, "cs_conflict")
	insertPendingStripeTicket(t, fx, eventID, "existing@example.test", "Existing", conflictingSession)
	fx.app.payments = &fakePaymentProvider{response: checkoutSessionResponse{ID: conflictingSession, URL: "https://checkout.example/conflict"}}
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("conflict"), "displayName": "Conflict"}, http.StatusInternalServerError)
	var status string
	if err := fx.app.db.QueryRow(t.Context(), `select a.status from payment_checkout_attempts a join tickets t on t.id = a.ticket_id where t.email = $1`, fx.email("conflict")).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "unknown" {
		t.Fatalf("binding conflict must remain reconcilable, got %q", status)
	}
}

func TestPaidReservationRetainsAmbiguousProviderFailureWithoutLeakingDetail(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 1, "fixed", 1500, "usd")
	fx.app.payments = failingCheckoutProvider{}
	slug := mustString(t, publishEvent(t, fx, mustString(t, event, "id")), "publicSlug")
	resp := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("timeout"), "displayName": "Timeout"}, http.StatusBadGateway)
	if strings.Contains(strings.ToLower(string(resp.Body)), "transport timeout") {
		t.Fatalf("provider detail leaked to client: %s", resp.Body)
	}
	var ticketStatus, attemptStatus, errorCode string
	if err := fx.app.db.QueryRow(t.Context(), `
		select t.payment_status, a.status, coalesce(a.last_error_code, '')
		from tickets t join payment_checkout_attempts a on a.ticket_id = t.id
		where t.event_id = $1
	`, mustString(t, event, "id")).Scan(&ticketStatus, &attemptStatus, &errorCode); err != nil {
		t.Fatal(err)
	}
	if ticketStatus != "pending" || attemptStatus != "unknown" || errorCode != "provider_unknown" {
		t.Fatalf("ambiguous checkout was not retained: ticket=%q attempt=%q error=%q", ticketStatus, attemptStatus, errorCode)
	}
}
