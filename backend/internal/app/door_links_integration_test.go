package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func createDoorLink(t *testing.T, fx lifecycleFixture, eventID, label string) (string, string) {
	t.Helper()
	result := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/door-links", map[string]any{"label": label, "validFrom": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), "validUntil": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}, http.StatusCreated)
	u, err := url.Parse(mustString(t, result.JSON, "url"))
	if err != nil {
		t.Fatal(err)
	}
	return mustString(t, result.JSON, "id"), u.Fragment
}
func linkRequest(app *App, eventID, token, action string, body any) *httptest.ResponseRecorder {
	method := http.MethodGet
	var payload []byte
	if body != nil {
		method = http.MethodPost
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, "/api/door-links/"+eventID+"/"+action, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(doorLinkHeader, token)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	return rec
}
func assertLinkStatus(t *testing.T, app *App, eventID, token, action string, body any, status int) *httptest.ResponseRecorder {
	t.Helper()
	rec := linkRequest(app, eventID, token, action, body)
	if rec.Code != status {
		t.Fatalf("door %s: status=%d want=%d body=%s", action, rec.Code, status, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("door cache header=%q", rec.Header().Get("Cache-Control"))
	}
	return rec
}

func TestDoorLinksRolesScopeWindowPrivacyAndAttribution(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Link Night", 50), "id")
	publishEvent(t, fx, eventID)
	_, memberID := memberIdentity(t, fx)
	path := "/api/events/" + eventID + "/door-links"
	request := map[string]any{"label": "Front door", "validFrom": time.Now().Add(-time.Minute), "validUntil": time.Now().Add(time.Hour)}
	for _, role := range []string{roleOrganizer, roleFinance, roleDoor, roleCrew, roleMember, roleParticipant} {
		if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role=$2 where id=$1`, memberID, role); err != nil {
			t.Fatal(err)
		}
		postJSON(t, fx.app, fx.memberCookie, path, request, 403)
		getJSON(t, fx.app, fx.memberCookie, path, 403)
	}
	other := newLifecycleFixture(t, fx.app)
	postJSON(t, fx.app, other.ownerCookie, path, request, 403)
	postJSON(t, fx.app, nil, path, request, 401)
	draft := mustString(t, createEvent(t, fx, "Draft", 1), "id")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+draft+"/door-links", request, 409)
	for _, req := range []map[string]any{
		{"label": "", "validFrom": time.Now(), "validUntil": time.Now().Add(time.Hour)},
		{"label": "x", "validFrom": time.Now(), "validUntil": time.Now().Add(8 * 24 * time.Hour)},
		{"label": "x", "validFrom": time.Now().Add(-time.Hour), "validUntil": time.Now().Add(-time.Minute)},
	} {
		postJSON(t, fx.app, fx.ownerCookie, path, req, 400)
	}
	id, token := createDoorLink(t, fx, eventID, "front-door-private-sentinel")
	secondID, second := createDoorLink(t, fx, eventID, "Side door")
	ticketID := insertTicketWithPaymentStatus(t, fx, eventID, "door-email-private@example.test", "Door Guest", "free", 0, "usd")
	code := ticketCode(t, fx, ticketID)
	for _, query := range []string{"", "ab", "%%%", "door-email-private"} {
		status := 200
		if len(query) < 3 {
			status = 400
		}
		rec := assertLinkStatus(t, fx.app, eventID, token, "tickets?query="+url.QueryEscape(query), nil, status)
		if status == 200 && rec.Body.String() != "[]\n" {
			t.Fatalf("enumeration or email lookup: %s", rec.Body.String())
		}
	}
	search := assertLinkStatus(t, fx.app, eventID, token, "tickets?query=Guest", nil, 200)
	for _, sentinel := range []string{"door-email-private", "amountCents", "currency", "paymentStatus", "ticketUrl", "workspaceId"} {
		if strings.Contains(search.Body.String(), sentinel) {
			t.Fatalf("door leaked %s", sentinel)
		}
	}
	assertLinkStatus(t, fx.app, draft, token, "tickets?query=Guest", nil, 403)
	assertLinkStatus(t, fx.app, eventID, "wrong-token-long-enough-to-pass-length-test", "tickets?query=Guest", nil, 403)
	// A different event ticket is refused without exposing its guest.
	otherTicket := insertTicketWithPaymentStatus(t, fx, draft, "other@example.test", "Other", "free", 0, "usd")
	assertLinkStatus(t, fx.app, eventID, token, "check-ins", map[string]string{"code": ticketCode(t, fx, otherTicket)}, 409)
	assertLinkStatus(t, fx.app, eventID, token, "check-ins", map[string]string{"code": code}, 200)
	assertLinkStatus(t, fx.app, eventID, second, "check-ins", map[string]string{"code": code}, 409)
	var actor *string
	var link string
	if err := fx.app.db.QueryRow(t.Context(), `select checked_in_by_person_id::text,checked_in_by_door_link_id::text from tickets where id=$1`, ticketID).Scan(&actor, &link); err != nil || actor != nil || link != id {
		t.Fatalf("actor=%v link=%s err=%v", actor, link, err)
	}
	var audits int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from audit_entries where action='ticket.checked_in' and metadata->>'doorLinkId'=$1`, id).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audit=%d err=%v", audits, err)
	}
	list := getJSON(t, fx.app, fx.ownerCookie, path, 200)
	if strings.Contains(list.Body, token) || !strings.Contains(list.Body, `"checkedInCount":1`) {
		t.Fatalf("list=%s", list.Body)
	}
	// Revoking is owner-only, and does not erase admissions.
	revoke := func(cookie *http.Cookie, linkID string, status int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodDelete, path+"/"+linkID, nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		fx.app.Handler().ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("revoke=%d: %s", rec.Code, rec.Body.String())
		}
	}
	revoke(fx.memberCookie, id, 403)
	revoke(fx.ownerCookie, id, 200)
	assertLinkStatus(t, fx.app, eventID, token, "tickets?query=Guest", nil, 403)
	// Future, exact-end expiry and cancellation all use the same denial.
	for _, update := range []string{`valid_from=clock_timestamp()+interval '1 minute'`, `valid_from=clock_timestamp()-interval '2 minutes',valid_until=clock_timestamp()`} {
		if _, err := fx.app.db.Exec(t.Context(), `update event_door_links set `+update+` where id=$1`, secondID); err != nil {
			t.Fatal(err)
		}
		rec := assertLinkStatus(t, fx.app, eventID, second, "tickets?query=Guest", nil, 403)
		if !strings.Contains(rec.Body.String(), "door link unavailable") {
			t.Fatal(rec.Body.String())
		}
	}
	_, active := createDoorLink(t, fx, eventID, "Cancelled link")
	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{"name": "Cancelled listing", "startsAt": "2026-11-14T20:00:00Z", "timezone": "UTC"}, 200)
	occurrenceID := mustString(t, occurrence.JSON, "id")
	if _, err := fx.app.db.Exec(t.Context(), `update event_occurrences set status='cancelled' where id=$1`, occurrenceID); err != nil {
		t.Fatal(err)
	}
	assertLinkStatus(t, fx.app, eventID, active, "tickets?query=Guest", nil, 403)
	if _, err := fx.app.db.Exec(t.Context(), `update event_occurrences set status='scheduled' where id=$1`, occurrenceID); err != nil {
		t.Fatal(err)
	}
	report := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, 200)
	if !strings.Contains(report.Body, `"label":"front-door-private-sentinel","count":1`) {
		t.Fatalf("report has no attributed admission: %s", report.Body)
	}

	if _, err := fx.app.db.Exec(t.Context(), `update events set status='end_of_night' where id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	assertLinkStatus(t, fx.app, eventID, active, "tickets?query=Guest", nil, 403)
}

func TestDoorLinksConcurrentCheckInAndRateLimit(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Door Race", 50), "id")
	publishEvent(t, fx, eventID)
	id, one := createDoorLink(t, fx, eventID, "One")
	_, two := createDoorLink(t, fx, eventID, "Two")
	ticketID := insertTicketWithPaymentStatus(t, fx, eventID, "race@example.test", "Guest", "free", 0, "usd")
	code := ticketCode(t, fx, ticketID)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for _, token := range []string{one, two} {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			results <- linkRequest(fx.app, eventID, token, "check-ins", map[string]string{"code": code}).Code
		}(token)
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for code := range results {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent=%v", counts)
	}
	// Database-level counting works across app processes and does not reset on restart.
	if _, err := fx.app.db.Exec(t.Context(), `update event_door_links set rate_count=60,rate_window=clock_timestamp() where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	assertLinkStatus(t, fx.app, eventID, one, "tickets?query=Guest", nil, 429)
	assertLinkStatus(t, fx.app, eventID, two, "tickets?query=Guest", nil, 200)
	if _, err := fx.app.db.Exec(t.Context(), `update event_door_links set rate_window=clock_timestamp()-interval '2 minutes' where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	assertLinkStatus(t, fx.app, eventID, one, "tickets?query=Guest", nil, 200)
	// Bound results and literal matching prevent wildcard roster downloads.
	for i := 0; i < 25; i++ {
		insertTicketWithPaymentStatus(t, fx, eventID, fmt.Sprintf("guest%d@example.test", i), "Guest", "free", 0, "usd")
	}
	rec := assertLinkStatus(t, fx.app, eventID, one, "tickets?query=Guest", nil, 200)
	var rows []doorTicketDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil || len(rows) != 20 {
		t.Fatalf("bounded search=%d err=%v", len(rows), err)
	}
}

func TestDoorLinkRechecksRevocationAfterTicketLock(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Door Revocation Race", 2), "id")
	publishEvent(t, fx, eventID)
	id, token := createDoorLink(t, fx, eventID, "Revoke while waiting")
	ticketID := insertTicketWithPaymentStatus(t, fx, eventID, "revoke@example.test", "Guest", "free", 0, "usd")
	code := ticketCode(t, fx, ticketID)
	blocker, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(t.Context(), `select id from tickets where id=$1 for update`, ticketID); err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() {
		result <- linkRequest(fx.app, eventID, token, "check-ins", map[string]string{"code": code}).Code
	}()
	deadline := time.Now().Add(10 * time.Second)
	blocked := false
	for time.Now().Before(deadline) {
		var n int
		if err := fx.app.db.QueryRow(t.Context(), `select count(*) from pg_stat_activity where datname=current_database() and wait_event_type='Lock' and query like '%from tickets%'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			blocked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("check-in never waited for ticket lock")
	}
	if _, err := fx.app.db.Exec(t.Context(), `update event_door_links set revoked_at=clock_timestamp() where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if err := blocker.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != 403 {
			t.Fatalf("revoked check-in=%d", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("check-in stuck")
	}
	var status string
	if err := fx.app.db.QueryRow(t.Context(), `select status from tickets where id=$1`, ticketID).Scan(&status); err != nil || status != "reserved" {
		t.Fatalf("status=%s err=%v", status, err)
	}
}
