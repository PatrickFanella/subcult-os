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

	"github.com/jackc/pgx/v5"

	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
)

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

type webhookTicketRow struct {
	ID                      string
	Email                   string
	Code                    string
	AmountCents             int
	Currency                string
	EventTitle              string
	PaymentStatus           string
	StripeCheckoutSessionID string
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
	if strings.TrimSpace(req.TicketID) == "" || strings.TrimSpace(req.EventID) == "" || strings.TrimSpace(req.EventTitle) == "" {
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
			"ticket_id": req.TicketID,
			"event_id":  req.EventID,
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
	params.SetIdempotencyKey("checkout-ticket-" + req.TicketID)

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
		return nil
	}

	var ticket webhookTicketRow
	err := tx.QueryRow(ctx, `
		select t.id, t.email, t.code, t.amount_cents, t.currency, e.title, t.payment_status, coalesce(t.stripe_checkout_session_id, '')
		from tickets t
		join events e on e.id = t.event_id
		where t.id = $1
		for update
	`, ticketID).Scan(&ticket.ID, &ticket.Email, &ticket.Code, &ticket.AmountCents, &ticket.Currency, &ticket.EventTitle, &ticket.PaymentStatus, &ticket.StripeCheckoutSessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if ticket.StripeCheckoutSessionID != session.ID || ticket.PaymentStatus != "pending" {
		return nil
	}
	if string(session.PaymentStatus) != string(stripe.CheckoutSessionPaymentStatusPaid) {
		return nil
	}
	if session.AmountTotal != int64(ticket.AmountCents) {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(string(session.Currency))) != strings.ToLower(strings.TrimSpace(ticket.Currency)) {
		return nil
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

	return nil
}

func (a *App) expireCheckoutSession(ctx context.Context, tx pgx.Tx, event stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return err
	}
	ticketID := strings.TrimSpace(session.Metadata["ticket_id"])
	if ticketID == "" || strings.TrimSpace(session.ID) == "" {
		return nil
	}

	var ticketIDOut string
	err := tx.QueryRow(ctx, `
		update tickets
		set payment_status = 'cancelled'
		where id = $1
		  and payment_status = 'pending'
		  and stripe_checkout_session_id = $2
		returning id
	`, ticketID, session.ID).Scan(&ticketIDOut)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	return a.audit(txCtx, "", "ticket.payment_expired", "ticket", ticketIDOut, map[string]any{
		"eventId":                 session.Metadata["event_id"],
		"stripeEventId":           event.ID,
		"stripeCheckoutSessionId": session.ID,
		"paymentStatus":           "cancelled",
	})
}
