package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v85"
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
