package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
)

type checkoutSessionRequest struct {
	TicketID               string
	EventID                string
	CheckoutAttemptID      string
	ProviderIdempotencyKey string
	EventTitle             string
	AmountCents            int
	Currency               string
	SuccessURL             string
	CancelURL              string
}

type checkoutSessionResponse struct {
	ID  string
	URL string
}

type paymentProvider interface {
	CreateCheckoutSession(ctx context.Context, req checkoutSessionRequest) (checkoutSessionResponse, error)
}

type webhookTicketRow struct {
	ID                      string
	EventID                 string
	Email                   string
	Code                    string
	AmountCents             int
	Currency                string
	EventTitle              string
	PaymentStatus           string
	StripeCheckoutSessionID string
	EventStatus             string
}

type stripePaymentProvider struct {
	client *stripe.Client
}

func newStripePaymentProvider(secretKey string) paymentProvider {
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		return nil
	}
	return &stripePaymentProvider{client: stripe.NewClient(secretKey)}
}

func (p *stripePaymentProvider) CreateCheckoutSession(ctx context.Context, req checkoutSessionRequest) (checkoutSessionResponse, error) {
	if p == nil || p.client == nil {
		return checkoutSessionResponse{}, fmt.Errorf("stripe provider unavailable")
	}
	if strings.TrimSpace(req.TicketID) == "" || strings.TrimSpace(req.EventID) == "" || strings.TrimSpace(req.CheckoutAttemptID) == "" || strings.TrimSpace(req.ProviderIdempotencyKey) == "" || strings.TrimSpace(req.EventTitle) == "" {
		return checkoutSessionResponse{}, fmt.Errorf("invalid checkout session request")
	}
	if req.AmountCents <= 0 {
		return checkoutSessionResponse{}, fmt.Errorf("invalid checkout amount")
	}
	if strings.TrimSpace(req.Currency) == "" || strings.TrimSpace(req.SuccessURL) == "" || strings.TrimSpace(req.CancelURL) == "" {
		return checkoutSessionResponse{}, fmt.Errorf("invalid checkout urls or currency")
	}

	params := &stripe.CheckoutSessionCreateParams{
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		PaymentMethodTypes: []*string{stripe.String("card")},
		SuccessURL:         stripe.String(req.SuccessURL),
		CancelURL:          stripe.String(req.CancelURL),
		ClientReferenceID:  stripe.String(req.TicketID),
		Metadata: map[string]string{
			"ticket_id":           req.TicketID,
			"event_id":            req.EventID,
			"checkout_attempt_id": req.CheckoutAttemptID,
		},
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:    stripe.String(strings.ToLower(strings.TrimSpace(req.Currency))),
				UnitAmount:  stripe.Int64(int64(req.AmountCents)),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{Name: stripe.String(req.EventTitle)},
			},
		}},
	}
	params.SetIdempotencyKey(req.ProviderIdempotencyKey)

	session, err := p.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return checkoutSessionResponse{}, err
	}
	return checkoutSessionResponse{ID: session.ID, URL: session.URL}, nil
}

func (a *App) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimSpace(a.config.StripeWebhookSecret)
	if secret == "" {
		writeError(w, http.StatusServiceUnavailable, "webhook secret unavailable")
		return
	}

	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "webhook payload too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), secret)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stripe webhook signature")
		return
	}
	if a.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if err := a.processStripeWebhookEvent(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "could not process webhook")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *App) processStripeWebhookEvent(ctx context.Context, event stripe.Event) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted, err := a.recordStripeWebhookEvent(ctx, tx, string(event.Type), event.ID)
	if err != nil || !inserted {
		return err
	}

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		if err := a.fulfillCheckoutSessionCompleted(ctx, tx, event); err != nil {
			return err
		}
	case stripe.EventTypeCheckoutSessionExpired:
		if err := a.expireCheckoutSession(ctx, tx, event); err != nil {
			return err
		}
	default:
		// ignored event types are acknowledged after recording idempotency
	}

	return tx.Commit(ctx)
}

func (a *App) recordStripeWebhookEvent(ctx context.Context, tx pgx.Tx, eventType string, eventID string) (bool, error) {
	var insertedID string
	err := tx.QueryRow(ctx, `
		insert into payment_webhook_events (id, provider, event_type)
		values ($1, 'stripe', $2)
		on conflict do nothing
		returning id
	`, eventID, eventType).Scan(&insertedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (a *App) fulfillCheckoutSessionCompleted(ctx context.Context, tx pgx.Tx, event stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return err
	}
	ticketID := strings.TrimSpace(session.Metadata["ticket_id"])
	if ticketID == "" || strings.TrimSpace(session.ID) == "" {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "missing_ticket_or_session")
	}
	if _, err := uuid.Parse(ticketID); err != nil {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "invalid_ticket")
	}

	var ticket webhookTicketRow
	err := tx.QueryRow(ctx, `
		select t.id, t.event_id, t.email, t.code, t.amount_cents, t.currency, e.title, t.payment_status, coalesce(t.stripe_checkout_session_id, ''), e.status
		from tickets t
		join events e on e.id = t.event_id
		where t.id = $1
		for update
	`, ticketID).Scan(&ticket.ID, &ticket.EventID, &ticket.Email, &ticket.Code, &ticket.AmountCents, &ticket.Currency, &ticket.EventTitle, &ticket.PaymentStatus, &ticket.StripeCheckoutSessionID, &ticket.EventStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "unknown_ticket")
		}
		return err
	}
	attemptID := strings.TrimSpace(session.Metadata["checkout_attempt_id"])
	if attemptID != "" {
		if _, err := uuid.Parse(attemptID); err != nil {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "invalid_attempt")
		}
	}
	if attemptID == "" {
		var hasAttempt bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from payment_checkout_attempts where ticket_id = $1)`, ticket.ID).Scan(&hasAttempt); err != nil {
			return err
		}
		if hasAttempt {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "missing_attempt")
		}
	}
	// Validate the immutable payment tuple before a recovery callback is allowed
	// to bind an unknown attempt to a provider session.
	if eventStatusIsClosed(ticket.EventStatus) {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "event_closed")
	}
	if string(session.PaymentStatus) != string(stripe.CheckoutSessionPaymentStatusPaid) {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "payment_not_paid")
	}
	if session.AmountTotal != int64(ticket.AmountCents) {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "amount_mismatch")
	}
	if strings.ToLower(strings.TrimSpace(string(session.Currency))) != strings.ToLower(strings.TrimSpace(ticket.Currency)) {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "currency_mismatch")
	}
	if attemptID != "" {
		var storedTicketID, provider, storedSession, status string
		err = tx.QueryRow(ctx, `
			select ticket_id, provider, coalesce(provider_session_id, ''), status
			from payment_checkout_attempts where id = $1 for update
		`, attemptID).Scan(&storedTicketID, &provider, &storedSession, &status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "unknown_attempt")
			}
			return err
		}
		if storedTicketID != ticket.ID || provider != "stripe" || (storedSession != "" && storedSession != session.ID) || (status != "ready" && status != "unknown" && status != "creating" && status != "fulfilled" && status != "expired") || strings.TrimSpace(session.Metadata["event_id"]) != ticket.EventID {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "attempt_mismatch")
		}
		if storedSession == "" {
			if ticket.StripeCheckoutSessionID != "" && ticket.StripeCheckoutSessionID != session.ID {
				return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "session_mismatch")
			}
			if _, err := tx.Exec(ctx, `update payment_checkout_attempts set provider_session_id = $2, status = 'ready', ready_at = coalesce(ready_at, now()), updated_at = now() where id = $1`, attemptID, session.ID); err != nil {
				return err
			}
			if ticket.StripeCheckoutSessionID == "" {
				if _, err := tx.Exec(ctx, `update tickets set stripe_checkout_session_id = $2 where id = $1 and stripe_checkout_session_id is null`, ticket.ID, session.ID); err != nil {
					return err
				}
				ticket.StripeCheckoutSessionID = session.ID
			}
		}
	}
	if ticket.PaymentStatus == "paid" && ticket.StripeCheckoutSessionID == session.ID {
		return a.setStripeWebhookOutcomeLink(ctx, tx, event.ID, "ignored", "duplicate_completed", ticket.ID, attemptID)
	}
	if ticket.PaymentStatus == "cancelled" && ticket.StripeCheckoutSessionID == session.ID {
		return a.setStripeWebhookOutcomeLink(ctx, tx, event.ID, "ignored", "expired_before_completed", ticket.ID, attemptID)
	}
	if ticket.StripeCheckoutSessionID != session.ID || !ticketJourneyIsPaymentPending(ticket.PaymentStatus) {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "ticket_not_pending")
	}

	paidAt := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		update tickets
		set payment_status = 'paid',
		    paid_at = $2
		where id = $1
		  and payment_status = 'pending'
		  and stripe_checkout_session_id = $3
	`, ticket.ID, paidAt, session.ID); err != nil {
		return err
	}
	if attemptID != "" {
		if _, err := tx.Exec(ctx, `update payment_checkout_attempts set status = 'fulfilled', terminal_at = now(), updated_at = now() where id = $1`, attemptID); err != nil {
			return err
		}
	}
	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	if err := a.audit(txCtx, "", "ticket.payment_completed", "ticket", ticket.ID, map[string]any{
		"eventId":                 session.Metadata["event_id"],
		"stripeEventId":           event.ID,
		"stripeCheckoutSessionId": session.ID,
		"email":                   ticket.Email,
		"amountCents":             ticket.AmountCents,
		"currency":                ticket.Currency,
		"paymentStatus":           "paid",
	}); err != nil {
		return err
	}

	ticketURL := a.publicTicketURL(ticket.Code)
	if err := a.enqueueEmail(txCtx, ticket.Email, "Your ticket for "+ticket.EventTitle, fmt.Sprintf("Your ticket for %s\n\nView your ticket: %s", ticket.EventTitle, ticketURL), "ticket", ticket.ID); err != nil {
		return err
	}

	return a.setStripeWebhookOutcomeLink(ctx, tx, event.ID, "fulfilled", "", ticket.ID, attemptID)
}

func (a *App) setStripeWebhookOutcome(ctx context.Context, tx pgx.Tx, eventID, outcome, reason string) error {
	return a.setStripeWebhookOutcomeLink(ctx, tx, eventID, outcome, reason, "", "")

}

func (a *App) setStripeWebhookOutcomeLink(ctx context.Context, tx pgx.Tx, eventID, outcome, reason, ticketID, attemptID string) error {
	_, err := tx.Exec(ctx, `
		update payment_webhook_events
		set outcome = $2, outcome_reason = nullif($3, ''),
		    ticket_id = nullif($4, '')::uuid,
		    checkout_attempt_id = nullif($5, '')::uuid
		where id = $1
	`, eventID, outcome, reason, ticketID, attemptID)
	return err
}

func (a *App) expireCheckoutSession(ctx context.Context, tx pgx.Tx, event stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return err
	}
	ticketID := strings.TrimSpace(session.Metadata["ticket_id"])
	if ticketID == "" || strings.TrimSpace(session.ID) == "" {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "missing_ticket_or_session")
	}
	if _, err := uuid.Parse(ticketID); err != nil {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "invalid_ticket")
	}
	var ticket webhookTicketRow
	err := tx.QueryRow(ctx, `
		select t.id, t.event_id, t.email, t.code, t.amount_cents, t.currency, e.title, t.payment_status, coalesce(t.stripe_checkout_session_id, ''), e.status
		from tickets t join events e on e.id = t.event_id
		where t.id = $1 for update
	`, ticketID).Scan(&ticket.ID, &ticket.EventID, &ticket.Email, &ticket.Code, &ticket.AmountCents, &ticket.Currency, &ticket.EventTitle, &ticket.PaymentStatus, &ticket.StripeCheckoutSessionID, &ticket.EventStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "unknown_ticket")
		}
		return err
	}
	attemptID := strings.TrimSpace(session.Metadata["checkout_attempt_id"])
	if attemptID != "" {
		if _, err := uuid.Parse(attemptID); err != nil {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "invalid_attempt")
		}
		if strings.ToLower(strings.TrimSpace(string(session.Currency))) != strings.ToLower(strings.TrimSpace(ticket.Currency)) || session.AmountTotal != int64(ticket.AmountCents) {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "payment_tuple_mismatch")
		}
	}
	if attemptID == "" {
		var hasAttempt bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from payment_checkout_attempts where ticket_id = $1)`, ticket.ID).Scan(&hasAttempt); err != nil {
			return err
		}
		if hasAttempt {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "missing_attempt")
		}
	} else {
		var storedTicketID, provider, storedSession, status string
		err = tx.QueryRow(ctx, `select ticket_id, provider, coalesce(provider_session_id, ''), status from payment_checkout_attempts where id = $1 for update`, attemptID).Scan(&storedTicketID, &provider, &storedSession, &status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "unknown_attempt")
			}
			return err
		}
		if storedTicketID != ticket.ID || provider != "stripe" || (storedSession != "" && storedSession != session.ID) || (status != "creating" && status != "unknown" && status != "ready" && status != "expired" && status != "fulfilled") || strings.TrimSpace(session.Metadata["event_id"]) != ticket.EventID {
			return a.setStripeWebhookOutcome(ctx, tx, event.ID, "anomalous", "attempt_mismatch")
		}
	}
	if ticket.PaymentStatus == "paid" && ticket.StripeCheckoutSessionID == session.ID {
		return a.setStripeWebhookOutcomeLink(ctx, tx, event.ID, "ignored", "completed_before_expired", ticket.ID, attemptID)
	}
	if ticket.PaymentStatus == "cancelled" && ticket.StripeCheckoutSessionID == session.ID {
		return a.setStripeWebhookOutcomeLink(ctx, tx, event.ID, "ignored", "duplicate_expired", ticket.ID, attemptID)
	}
	if !ticketJourneyIsPaymentPending(ticket.PaymentStatus) || (ticket.StripeCheckoutSessionID != "" && ticket.StripeCheckoutSessionID != session.ID) {
		return a.setStripeWebhookOutcome(ctx, tx, event.ID, "ignored", "ticket_not_pending")
	}
	if _, err := tx.Exec(ctx, `update tickets set stripe_checkout_session_id = $2, payment_status = 'cancelled' where id = $1`, ticket.ID, session.ID); err != nil {
		return err
	}
	if attemptID != "" {
		if _, err := tx.Exec(ctx, `update payment_checkout_attempts set provider_session_id = $2, status = 'expired', terminal_at = now(), updated_at = now() where id = $1`, attemptID, session.ID); err != nil {
			return err
		}
	}
	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := a.audit(txCtx, "", "ticket.payment_expired", "ticket", ticket.ID, map[string]any{
		"eventId":                 session.Metadata["event_id"],
		"stripeEventId":           event.ID,
		"stripeCheckoutSessionId": session.ID,
		"paymentStatus":           "cancelled",
	}); err != nil {
		return err
	}
	return a.setStripeWebhookOutcomeLink(ctx, tx, event.ID, "expired", "", ticket.ID, attemptID)
}
