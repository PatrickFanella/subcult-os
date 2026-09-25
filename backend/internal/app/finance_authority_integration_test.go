package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFinancePermissionGuardsSettlementReadsAndOperations(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Finance Night", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	for _, path := range []string{"/api/events/" + eventID + "/settlement", "/api/events/" + eventID + "/report"} {
		getJSON(t, fx.app, fx.memberCookie, path, http.StatusForbidden)
	}
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{"amountCents": 100, "label": "Member", "reason": "denied"}, http.StatusForbidden)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement/finalize", map[string]any{}, http.StatusForbidden)

	memberID := memberRowIDOnly(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'finance' where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement", http.StatusOK)
	getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/report", http.StatusOK)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{"amountCents": 100, "label": "Finance", "reason": "authorized"}, http.StatusOK)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement/finalize", map[string]any{}, http.StatusOK)
}

func TestOperationalPatchDoesNotRestoreStalePricingAfterFinanceCommit(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Race Night", 2)
	eventID := mustString(t, event, "id")
	blocker, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(t.Context(), `select id from events where id = $1 for update`, eventID); err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() {
		body, _ := json.Marshal(map[string]any{"locationDisplay": "Operational room"})
		req := httptest.NewRequest(http.MethodPatch, "/api/events/"+eventID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(fx.memberCookie)
		rec := httptest.NewRecorder()
		fx.app.Handler().ServeHTTP(rec, req)
		result <- rec.Code
	}()
	deadline := time.Now().Add(10 * time.Second)
	blocked := false
	for time.Now().Before(deadline) {
		var waiting int
		if err := fx.app.db.QueryRow(t.Context(), `select count(*) from pg_stat_activity where datname=current_database() and wait_event_type='Lock' and query like '%update events%'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			blocked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("timed out waiting for the operational PATCH to block on the event update")
	}
	if _, err := blocker.Exec(t.Context(), `update events set pricing_mode='fixed', ticket_price_cents=1500, ticket_currency='usd' where id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	if err := blocker.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != http.StatusOK {
			t.Fatalf("operational patch status=%d", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for operational patch")
	}
	var mode, currency string
	var price int
	if err := fx.app.db.QueryRow(t.Context(), `select pricing_mode, ticket_price_cents, ticket_currency from events where id=$1`, eventID).Scan(&mode, &price, &currency); err != nil {
		t.Fatal(err)
	}
	if mode != "fixed" || price != 1500 || currency != "usd" {
		t.Fatalf("operational patch restored stale pricing: %s %d %s", mode, price, currency)
	}
}

func TestPricingWritesRequireFinanceButFreeOperationalEditsRemainAvailable(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Free Night", 2)
	eventID := mustString(t, event, "id")
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Member free", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Free.", "locationDisplay": "Warehouse", "ticketAllocation": 2, "pricingMode": "free"}, http.StatusOK)
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Member paid", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Paid.", "locationDisplay": "Warehouse", "ticketAllocation": 2, "pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusForbidden)
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"locationDisplay": "New room"}, http.StatusOK)
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusForbidden)
	memberID := memberRowIDOnly(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'finance' where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusOK)

	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Paid by finance", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Paid.", "locationDisplay": "Warehouse", "ticketAllocation": 2, "pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusOK)
}

func TestFinanceBoundaryDeniesEveryNonFinanceRoleAndInactiveMembership(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Finance Night", 1, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	memberID := memberRowIDOnly(t, fx)
	for _, role := range []string{roleMember, roleOrganizer, roleDoor, roleCrew} {
		if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = $2, revoked_at = null, expires_at = null where id = $1`, memberID, role); err != nil {
			t.Fatal(err)
		}
		getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement", http.StatusForbidden)
		getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/report", http.StatusForbidden)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'finance', revoked_at = now() where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement", http.StatusForbidden)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at = null, expires_at = now() - interval '1 second' where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/report", http.StatusForbidden)
	other := newLifecycleFixture(t, fx.app)
	getJSON(t, fx.app, other.ownerCookie, "/api/events/"+eventID+"/settlement", http.StatusForbidden)
}
