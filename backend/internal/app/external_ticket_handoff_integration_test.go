package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func externalTicketHandoffPath(eventID, occurrenceID string) string {
	return "/api/events/" + eventID + "/occurrences/" + occurrenceID + "/external-ticket-handoff"
}

func externalTicketHandoffMemberRowID(t *testing.T, fx lifecycleFixture) string {
	t.Helper()
	var id string
	if err := fx.app.db.QueryRow(t.Context(), `
		select wm.id from workspace_members wm
		join people p on p.id = wm.person_id
		where wm.workspace_id = $1 and p.email = $2
	`, fx.workspaceID, fx.email("member")).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func newExternalTicketHandoffFixture(t *testing.T) (lifecycleFixture, string, string) {
	t.Helper()
	fx := newCulturalFixture(t)
	fx.app.config.ExternalTicketAllowedHosts = []string{"tickets.example.test"}
	event := createEvent(t, fx, "External handoff fixture", 10)
	eventID := mustString(t, event, "id")
	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"name": "External handoff occurrence", "startsAt": "2026-10-01T20:00:00Z",
	}, http.StatusOK)
	return fx, eventID, mustString(t, occurrence.JSON, "id")
}

func TestExternalTicketHandoffConfiguresOnlyTheCanonicalOccurrence(t *testing.T) {
	fx, eventID, occurrenceID := newExternalTicketHandoffFixture(t)
	path := externalTicketHandoffPath(eventID, occurrenceID)
	configured := doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/events/signal-night?ref=public", "providerLabel": "Example Tickets", "expectedUpdatedAt": "",
	}, http.StatusOK)
	if field(t, configured.JSON, "purchaseUrl") != "https://tickets.example.test/events/signal-night?ref=public" || field(t, configured.JSON, "providerLabel") != "Example Tickets" {
		t.Fatalf("configured handoff = %#v", configured.JSON)
	}
	if field(t, configured.JSON, "visibility") != "private" {
		t.Fatalf("handoff visibility = %#v", configured.JSON)
	}
	getJSON(t, fx.app, fx.ownerCookie, path, http.StatusOK)

	// The configuration cannot turn a private occurrence into a public
	// discovery item. That route reads only at_projection_records and remains
	// empty without a separately projected public occurrence.
	list := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences", http.StatusOK)
	if entries, ok := list.JSON.([]any); !ok || len(entries) != 0 {
		t.Fatalf("private handoff leaked into discovery: %#v", list.JSON)
	}
	for _, table := range []string{"tickets", "payment_checkout_attempts", "payment_webhook_events"} {
		var count int
		if err := fx.app.db.QueryRow(t.Context(), "select count(*) from "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("handoff configuration touched %s: count=%d err=%v", table, count, err)
		}
	}
}

func TestExternalTicketHandoffRejectsUnapprovedAndUnsafeDestinations(t *testing.T) {
	fx, eventID, occurrenceID := newExternalTicketHandoffFixture(t)
	path := externalTicketHandoffPath(eventID, occurrenceID)
	doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/event", "providerLabel": "Example Tickets",
	}, http.StatusBadRequest)
	for _, purchaseURL := range []string{
		"http://tickets.example.test/event", "https://tickets.example.test.evil.test/event",
		"https://tickets.example.test@evil.example.test/event", "https://tickets.example.test:8443/event",
		"https://tickets.example.test/event", // exercised after allowlist removal below
	} {
		if purchaseURL == "https://tickets.example.test/event" {
			fx.app.config.ExternalTicketAllowedHosts = nil // empty is deny-by-default
		}
		doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
			"purchaseUrl": purchaseURL, "providerLabel": "Example Tickets", "expectedUpdatedAt": "",
		}, http.StatusBadRequest)
		fx.app.config.ExternalTicketAllowedHosts = []string{"tickets.example.test"}
	}
	doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/event", "providerLabel": " Example Tickets\n", "expectedUpdatedAt": "",
	}, http.StatusBadRequest)
	doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/event", "providerLabel": "Example\nTickets", "expectedUpdatedAt": "",
	}, http.StatusBadRequest)
}

func TestExternalTicketHandoffRechecksRevocationAfterOccurrenceLock(t *testing.T) {
	fx, eventID, occurrenceID := newExternalTicketHandoffFixture(t)
	path := externalTicketHandoffPath(eventID, occurrenceID)
	lockTx, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer lockTx.Rollback(t.Context())
	if _, err := lockTx.Exec(t.Context(), `select id from event_occurrences where id=$1 for update`, occurrenceID); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"purchaseUrl": "https://tickets.example.test/event", "providerLabel": "Example Tickets", "expectedUpdatedAt": ""})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() {
		req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(fx.ownerCookie)
		rec := httptest.NewRecorder()
		fx.app.Handler().ServeHTTP(rec, req)
		result <- rec.Code
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		err := fx.app.db.QueryRow(t.Context(), `
			select exists (
				select 1 from pg_stat_activity
				where datname = current_database() and wait_event_type = 'Lock'
				  and query like '%event_occurrences where id=$1 and event_id=$2 for update%'
			)
		`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("handoff request did not block on occurrence lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	ownerID := ownerMemberRowID(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=clock_timestamp() where id=$1`, ownerID); err != nil {
		t.Fatal(err)
	}
	if err := lockTx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if status := <-result; status != http.StatusForbidden {
		t.Fatalf("revoked blocked request = %d, want 403", status)
	}
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from occurrence_external_ticket_handoffs where occurrence_id=$1`, occurrenceID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("revoked request saved handoff: count=%d err=%v", count, err)
	}
}

func TestExternalTicketHandoffRequiresOwnerOrOrganizerAndCurrentMembership(t *testing.T) {
	fx, eventID, occurrenceID := newExternalTicketHandoffFixture(t)
	path := externalTicketHandoffPath(eventID, occurrenceID)
	payload := map[string]any{"purchaseUrl": "https://tickets.example.test/event", "providerLabel": "Example Tickets", "expectedUpdatedAt": ""}
	doJSON(t, http.MethodPut, fx.app, fx.memberCookie, path, payload, http.StatusForbidden)

	memberID := externalTicketHandoffMemberRowID(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role=$2, revoked_at=null, expires_at=null where id=$1`, memberID, roleOrganizer); err != nil {
		t.Fatal(err)
	}
	doJSON(t, http.MethodPut, fx.app, fx.memberCookie, path, payload, http.StatusOK)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=clock_timestamp() where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	doJSON(t, http.MethodPut, fx.app, fx.memberCookie, path, payload, http.StatusForbidden)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=null, expires_at=clock_timestamp()-interval '1 second' where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, path, http.StatusForbidden)
}

func TestExternalTicketHandoffPreservesRevisionCAS(t *testing.T) {
	fx, eventID, occurrenceID := newExternalTicketHandoffFixture(t)
	path := externalTicketHandoffPath(eventID, occurrenceID)
	first := doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/first", "providerLabel": "Example Tickets", "expectedUpdatedAt": "",
	}, http.StatusOK)
	revision := mustString(t, first.JSON, "updatedAt")
	updated := doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/second", "providerLabel": "Example Tickets", "expectedUpdatedAt": revision,
	}, http.StatusOK)
	if mustString(t, updated.JSON, "updatedAt") == revision {
		t.Fatal("handoff revision did not advance")
	}
	doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/stale", "providerLabel": "Example Tickets", "expectedUpdatedAt": revision,
	}, http.StatusConflict)

	var purchaseURL string
	if err := fx.app.db.QueryRow(t.Context(), `select purchase_url from occurrence_external_ticket_handoffs where occurrence_id=$1`, occurrenceID).Scan(&purchaseURL); err != nil || purchaseURL != "https://tickets.example.test/second" {
		t.Fatalf("stale update changed handoff: url=%q err=%v", purchaseURL, err)
	}
}

func TestExternalTicketHandoffStopsPreviewWhenAllowlistChanges(t *testing.T) {
	fx, eventID, occurrenceID := newExternalTicketHandoffFixture(t)
	path := externalTicketHandoffPath(eventID, occurrenceID)
	doJSON(t, http.MethodPut, fx.app, fx.ownerCookie, path, map[string]any{
		"purchaseUrl": "https://tickets.example.test/first", "providerLabel": "Example Tickets", "expectedUpdatedAt": "",
	}, http.StatusOK)
	fx.app.config.ExternalTicketAllowedHosts = nil
	getJSON(t, fx.app, fx.ownerCookie, path, http.StatusConflict)
}

func TestExternalTicketAllowedHostsConfigValidation(t *testing.T) {
	if got := parseExternalTicketAllowedHosts(" Tickets.Example.Test, tickets.example.test "); strings.Join(got, ",") != "tickets.example.test,tickets.example.test" {
		t.Fatalf("parsed hosts = %#v", got)
	}
	for _, host := range []string{"tickets.example.test/path", "tickets.example.test:443", "user@tickets.example.test"} {
		if err := (Config{ExternalTicketAllowedHosts: []string{host}}).Validate(); err == nil {
			t.Fatalf("invalid allowlist host accepted: %q", host)
		}
	}
	if err := (Config{ExternalTicketAllowedHosts: []string{"tickets.example.test"}}).Validate(); err != nil {
		t.Fatal(err)
	}
}
