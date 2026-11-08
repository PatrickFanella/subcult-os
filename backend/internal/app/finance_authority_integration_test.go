package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
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
	// Operational edits belong to coordinators; an organizer has no finance.
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'organizer' where id = $1`, memberRowIDOnly(t, fx)); err != nil {
		t.Fatal(err)
	}
	blocker, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(t.Context(), `select id from events where id = $1 for update`, eventID); err != nil {
		t.Fatal(err)
	}
	var blockerPID int
	if err := blocker.QueryRow(t.Context(), `select pg_backend_pid()`).Scan(&blockerPID); err != nil {
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
		if err := fx.app.db.QueryRow(t.Context(), `select count(*) from pg_stat_activity where datname=current_database() and $1=any(pg_blocking_pids(pid))`, blockerPID).Scan(&waiting); err != nil {
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
	// Crew cannot create or edit events at all.
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Crew free", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Free.", "locationDisplay": "Warehouse", "ticketAllocation": 2, "pricingMode": "free"}, http.StatusForbidden)
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"locationDisplay": "Crew room"}, http.StatusForbidden)
	// An organizer edits events but not pricing.
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'organizer' where id = $1`, memberRowIDOnly(t, fx)); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Member free", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Free.", "locationDisplay": "Warehouse", "ticketAllocation": 2, "pricingMode": "free"}, http.StatusOK)
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Member paid", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Paid.", "locationDisplay": "Warehouse", "ticketAllocation": 2, "pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusForbidden)
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"locationDisplay": "New room"}, http.StatusOK)
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusForbidden)
	memberID := memberRowIDOnly(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'finance' where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusOK)
	// Finance prices events; planning them stays with coordinators.
	patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"locationDisplay": "Finance room"}, http.StatusForbidden)
	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Paid by finance", "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Paid.", "locationDisplay": "Warehouse", "ticketAllocation": 2, "pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}, http.StatusForbidden)
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

// Pause the UPDATE after its early permission checks, then change the member's
// role before the final authorization check. The gate exists only in the
// disposable test database and applies to this one event.
func TestPricingPatchRechecksFinanceAfterOwnerDemotion(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(fmt.Sprintf("mixed_operational_edit_%t", mixed), func(t *testing.T) {
			fx := newLifecycleFixture(t)
			eventID := mustString(t, createEvent(t, fx, "Demotion window", 2), "id")
			memberID := memberRowIDOnly(t, fx)
			if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role='owner' where id=$1`, memberID); err != nil {
				t.Fatal(err)
			}
			name := pgx.Identifier{"test_pricing_gate_" + strings.ReplaceAll(eventID, "-", "")}.Sanitize()
			if _, err := fx.app.db.Exec(t.Context(), fmt.Sprintf(`create function %s() returns trigger language plpgsql as $$ begin perform pg_advisory_xact_lock(hashtextextended(NEW.id::text,42)); return NEW; end $$`, name)); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := fx.app.db.Exec(context.Background(), fmt.Sprintf(`drop trigger if exists %s on events; drop function if exists %s()`, name, name)); err != nil {
					t.Error(err)
				}
			}()
			if _, err := fx.app.db.Exec(t.Context(), fmt.Sprintf(`create trigger %s before update on events for each row when (NEW.id='%s'::uuid) execute function %s()`, name, eventID, name)); err != nil {
				t.Fatal(err)
			}
			blocker, err := fx.app.db.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = blocker.Rollback(context.Background()) }()
			if _, err := blocker.Exec(t.Context(), `select pg_advisory_xact_lock(hashtextextended($1::text,42))`, eventID); err != nil {
				t.Fatal(err)
			}
			var blockerPID int
			if err := blocker.QueryRow(t.Context(), `select pg_backend_pid()`).Scan(&blockerPID); err != nil {
				t.Fatal(err)
			}
			result := make(chan int, 1)
			go func() {
				body := map[string]any{"pricingMode": "fixed", "ticketPriceCents": 1500, "ticketCurrency": "usd"}
				if mixed {
					body["locationDisplay"] = "Updated by demoted owner"
				}
				payload, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPatch, "/api/events/"+eventID, bytes.NewReader(payload))
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
				if err := fx.app.db.QueryRow(t.Context(), `select count(*) from pg_stat_activity where datname=current_database() and $1=any(pg_blocking_pids(pid))`, blockerPID).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting > 0 {
					blocked = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !blocked {
				t.Fatal("pricing UPDATE did not reach the permission-check gate")
			}
			if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role='organizer' where id=$1`, memberID); err != nil {
				t.Fatal(err)
			}
			if err := blocker.Commit(t.Context()); err != nil {
				t.Fatal(err)
			}
			select {
			case status := <-result:
				if status != http.StatusForbidden {
					t.Errorf("demoted pricing writer returned %d, want 403", status)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("pricing PATCH did not finish")
			}
			var mode, location string
			var price, updates int
			if err := fx.app.db.QueryRow(t.Context(), `select pricing_mode,ticket_price_cents,location_display from events where id=$1`, eventID).Scan(&mode, &price, &location); err != nil {
				t.Fatal(err)
			}
			if mode != "free" || price != 0 || location == "Updated by demoted owner" {
				t.Errorf("demoted writer committed event changes: mode=%s price=%d location=%s", mode, price, location)
			}
			if err := fx.app.db.QueryRow(t.Context(), `select count(*) from audit_entries where subject_id=$1 and action='event.updated'`, eventID).Scan(&updates); err != nil {
				t.Fatal(err)
			}
			if updates != 0 {
				t.Errorf("denied pricing write recorded %d success audits", updates)
			}
		})
	}
}
