package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoorPermissionAndAdmissionDTO(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEventWithPricing(t, fx, "Door Authority", 3, "fixed", 1500, "usd"), "id")
	publishEvent(t, fx, eventID)
	ticketID := insertTicketWithPaymentStatus(t, fx, eventID, "door-private@example.test", "Door Guest", "paid", 1500, "usd")
	code := ticketCode(t, fx, ticketID)
	_, memberID := memberIdentity(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `
		insert into contacts (workspace_id, display_name, email, notes, created_by_person_id)
		values ($1, 'Door private contact', 'door-contact-private@example.test', 'door-contact-private-note', $2)
	`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
		t.Fatal(err)
	}
	searchPath := "/api/events/" + eventID + "/door/tickets?query=door-private"
	checkInPath := "/api/events/" + eventID + "/door/check-ins"

	getJSON(t, fx.app, fx.ownerCookie, searchPath, http.StatusOK)
	getJSON(t, fx.app, fx.memberCookie, searchPath, http.StatusForbidden)
	postJSON(t, fx.app, fx.memberCookie, checkInPath, map[string]any{"code": code}, http.StatusForbidden)

	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'door' where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	search := getJSON(t, fx.app, fx.memberCookie, searchPath, http.StatusOK)
	if got := search.Header.Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("door search cache-control = %q", got)
	}
	items := search.JSON.([]any)
	if len(items) != 1 {
		t.Fatalf("door search = %#v", search.JSON)
	}
	entry := mustObject(t, items[0])
	if entry["code"] != code || entry["displayName"] != "Door Guest" || entry["admissionEligible"] != true || entry["status"] != "reserved" {
		t.Fatalf("door ticket = %#v", entry)
	}
	for _, forbidden := range []string{"email", "amountCents", "currency", "paymentStatus", "eventId", "ticketUrl"} {
		if _, exists := entry[forbidden]; exists {
			t.Fatalf("door ticket exposed %s: %#v", forbidden, entry)
		}
	}
	if strings.Contains(search.Body, "door-private@example.test") || strings.Contains(search.Body, "door-contact-private@example.test") || strings.Contains(search.Body, "door-contact-private-note") {
		t.Fatalf("door search leaked a private value: %s", search.Body)
	}
	checkedIn := postJSON(t, fx.app, fx.memberCookie, checkInPath, map[string]any{"code": code}, http.StatusOK)
	if got := checkedIn.Header.Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("door check-in cache-control = %q", got)
	}
	checkedInTicket := mustObject(t, checkedIn.JSON)
	if checkedInTicket["status"] != "checked_in" || checkedInTicket["checkedInAt"] == "" || checkedInTicket["admissionEligible"] != true {
		t.Fatalf("check-in = %#v", checkedIn.JSON)
	}
	if _, exists := checkedInTicket["email"]; exists {
		t.Fatalf("check-in exposed email: %#v", checkedInTicket)
	}

	for _, role := range []string{roleMember, roleOrganizer, roleFinance, roleCrew} {
		if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = $2, revoked_at = null, expires_at = null where id = $1`, memberID, role); err != nil {
			t.Fatal(err)
		}
		getJSON(t, fx.app, fx.memberCookie, searchPath, http.StatusForbidden)
		postJSON(t, fx.app, fx.memberCookie, checkInPath, map[string]any{"code": code}, http.StatusForbidden)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'door', revoked_at = now() where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, searchPath, http.StatusForbidden)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at = null, expires_at = now() - interval '1 second' where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.memberCookie, checkInPath, map[string]any{"code": code}, http.StatusForbidden)

	other := newLifecycleFixture(t, fx.app)
	getJSON(t, fx.app, other.ownerCookie, searchPath, http.StatusForbidden)

	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'door', expires_at = null where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	otherEventID := mustString(t, createEvent(t, fx, "Other Door Event", 1), "id")
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+otherEventID+"/door/check-ins", map[string]any{"code": code}, http.StatusConflict)
}

func TestDoorCheckInRechecksActivePermissionAfterTicketLock(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Door Permission Race", 2), "id")
	publishEvent(t, fx, eventID)
	ticketID := insertTicketWithPaymentStatus(t, fx, eventID, "door-race@example.test", "Door Race", "free", 0, "usd")
	code := ticketCode(t, fx, ticketID)
	_, memberID := memberIdentity(t, fx)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'door' where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}

	blocker, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(t.Context(), `select id from tickets where code = $1 for update`, code); err != nil {
		t.Fatal(err)
	}

	result := make(chan int, 1)
	go func() {
		body, _ := json.Marshal(map[string]any{"code": code})
		req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/door/check-ins", bytes.NewReader(body))
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
		if err := fx.app.db.QueryRow(t.Context(), `
			select count(*)
			from pg_stat_activity
			where datname = current_database()
			  and wait_event_type = 'Lock'
			  and query like '%from tickets%'
		`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			blocked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("timed out waiting for door check-in to block on the ticket lock")
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at = now() where id = $1`, memberID); err != nil {
		t.Fatal(err)
	}
	if err := blocker.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}

	select {
	case status := <-result:
		if status != http.StatusForbidden {
			t.Fatalf("check-in after revoked door access status=%d", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for blocked door check-in")
	}
	var status string
	if err := fx.app.db.QueryRow(t.Context(), `select status from tickets where code = $1`, code).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "reserved" {
		t.Fatalf("revoked door access checked in ticket: %s", status)
	}
}

func ticketCode(t *testing.T, fx lifecycleFixture, ticketID string) string {
	t.Helper()
	var code string
	if err := fx.app.db.QueryRow(t.Context(), `select code from tickets where id = $1`, ticketID).Scan(&code); err != nil {
		t.Fatal(err)
	}
	return code
}
