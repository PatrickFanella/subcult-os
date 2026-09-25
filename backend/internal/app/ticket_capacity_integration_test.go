package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestTicketReservationsDoNotOverbookWhenRequestsQueueOnTheSameEvent holds an
// event row while two admission requests begin. Both requests therefore take
// their statement snapshot before the row lock is released. Once the lock is
// available, exactly one request may consume the final ticket.
func TestTicketReservationsDoNotOverbookWhenRequestsQueueOnTheSameEvent(t *testing.T) {
	tests := []struct {
		name        string
		pricingMode string
		priceCents  int
		prepare     func(t *testing.T, fx lifecycleFixture, eventID, slug string)
		path        func(eventID, slug string) string
		cookie      func(fx lifecycleFixture) *http.Cookie
	}{
		{
			name:        "free public reservation",
			pricingMode: "free",
			prepare:     func(*testing.T, lifecycleFixture, string, string) {},
			path:        func(_ string, slug string) string { return "/api/public/events/" + slug + "/reservations" },
			cookie:      func(lifecycleFixture) *http.Cookie { return nil },
		},
		{
			name:        "paid public reservation",
			pricingMode: "fixed",
			priceCents:  1500,
			prepare: func(_ *testing.T, fx lifecycleFixture, _, _ string) {
				fx.app.payments = &capacityPaymentProvider{}
			},
			path:   func(_ string, slug string) string { return "/api/public/events/" + slug + "/paid-reservations" },
			cookie: func(lifecycleFixture) *http.Cookie { return nil },
		},
		{
			name:        "operator test ticket",
			pricingMode: "free",
			prepare:     func(*testing.T, lifecycleFixture, string, string) {},
			path:        func(eventID, _ string) string { return "/api/events/" + eventID + "/test-ticket" },
			cookie:      func(fx lifecycleFixture) *http.Cookie { return fx.ownerCookie },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newLifecycleFixture(t)
			event := createEventWithPricing(t, fx, "Capacity One", 1, tt.pricingMode, tt.priceCents, "usd")
			eventID := mustString(t, event, "id")
			slug := ""
			if tt.path(eventID, slug) != "/api/events/"+eventID+"/test-ticket" {
				slug = mustString(t, publishEvent(t, fx, eventID), "publicSlug")
			}
			tt.prepare(t, fx, eventID, slug)

			blocker, err := fx.app.db.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = blocker.Rollback(context.Background()) }()
			if _, err := blocker.Exec(t.Context(), `select id from events where id = $1 for update`, eventID); err != nil {
				t.Fatal(err)
			}

			path := tt.path(eventID, slug)
			results := make(chan ticketCapacityResponse, 2)
			for index := range 2 {
				go func(index int) {
					results <- requestTicketCapacityReservation(fx.app, tt.cookie(fx), path, map[string]any{
						"email":       fmt.Sprintf("capacity-%d-%s@example.test", index, fx.suffix),
						"displayName": "Capacity Guest",
					})
				}(index)
			}

			waitForTicketReservationLockers(t, fx, 2)
			if err := blocker.Commit(t.Context()); err != nil {
				t.Fatal(err)
			}

			statuses := ticketCapacityResponseStatuses(t, results)
			if statuses[http.StatusOK] != 1 || statuses[http.StatusConflict] != 1 {
				t.Fatalf("reservation statuses = %#v, want one 200 and one 409", statuses)
			}

			var reserved int
			if err := fx.app.db.QueryRow(t.Context(), `
				select count(*) from tickets where event_id = $1 and payment_status <> 'cancelled'
			`, eventID).Scan(&reserved); err != nil {
				t.Fatal(err)
			}
			if reserved != 1 {
				t.Fatalf("non-cancelled tickets = %d, want 1", reserved)
			}
		})
	}
}

type capacityPaymentProvider struct {
	mu   sync.Mutex
	next int
}

func (p *capacityPaymentProvider) CreateCheckoutSession(_ context.Context, _ checkoutSessionRequest) (checkoutSessionResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next++
	return checkoutSessionResponse{
		ID:  fmt.Sprintf("cs_test_capacity_%d", p.next),
		URL: fmt.Sprintf("https://checkout.example.test/capacity/%d", p.next),
	}, nil
}

type ticketCapacityResponse struct {
	status int
	body   string
}

func requestTicketCapacityReservation(app *App, cookie *http.Cookie, path string, payload any) ticketCapacityResponse {
	body, err := json.Marshal(payload)
	if err != nil {
		return ticketCapacityResponse{body: err.Error()}
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Host = "public.test"
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	return ticketCapacityResponse{status: rec.Code, body: rec.Body.String()}
}

func waitForTicketReservationLockers(t *testing.T, fx lifecycleFixture, want int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		err := fx.app.db.QueryRow(t.Context(), `
			select count(*)
			from pg_stat_activity
			where datname = current_database()
			  and wait_event_type = 'Lock'
			  and query like '%from events e%'
			  and query like '%for update%'
		`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for both ticket reservations to block on the event row")
}

func ticketCapacityResponseStatuses(t *testing.T, results <-chan ticketCapacityResponse) map[int]int {
	t.Helper()
	statuses := make(map[int]int)
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for range 2 {
		select {
		case result := <-results:
			statuses[result.status]++
		case <-deadline.C:
			t.Fatalf("timed out waiting for reservation responses: %#v", statuses)
		}
	}
	return statuses
}
