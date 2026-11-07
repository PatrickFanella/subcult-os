package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestParticipantPortalShowsOnlyCurrentPersonAssignmentsAndCommitments(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Participant Portal Event", 10), "id")
	memberPersonID, memberRowID := memberIdentity(t, fx)
	ownerID := ownerPersonID(t, fx)

	var staffingID string
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into event_staffing_items (event_id, title, kind, notes, participant_requirements, starts_at, ends_at, assigned_person_id, status, created_by_person_id)
		values ($1, 'Soundcheck', 'shift', 'operator-only staffing note', 'Bring a DI and arrive 15 minutes early.', '2026-07-01T18:00:00Z', '2026-07-01T19:00:00Z', $2, 'assigned', $3)
		returning id
	`, eventID, memberPersonID, ownerID).Scan(&staffingID); err != nil {
		t.Fatal(err)
	}

	var roleID, applicationID string
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into event_roles (event_id, name, description, capacity, "public", active, created_by_person_id)
		values ($1, 'Performer', '', 1, true, true, $2) returning id
	`, eventID, ownerID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into event_role_applications (event_id, role_id, applicant_name, applicant_email, message, status)
		values ($1, $2, 'Door', $3, 'application-private-message', 'accepted') returning id
	`, eventID, roleID, fx.email("member")).Scan(&applicationID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `
		insert into event_staffing_items (event_id, title, kind, notes, participant_requirements, assigned_application_id, status, created_by_person_id)
		values ($1, 'Application-only assignment', 'task', 'application-only private note', 'application-only requirement', $2, 'assigned', $3)
	`, eventID, applicationID, ownerID); err != nil {
		t.Fatal(err)
	}

	var contactID string
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into contacts (workspace_id, display_name, email, notes, created_by_person_id)
		values ($1, 'Private Contact', 'contact-private@example.test', 'contact-private-note', $2)
		returning id
	`, fx.workspaceID, ownerID).Scan(&contactID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `
		insert into commitments (workspace_id, event_id, contact_id, title, description, due_at, status, owner_person_id, created_by_person_id)
		values ($1, $2, $3, 'Bring cables', 'commitment-private-description', '2026-07-01T17:00:00Z', 'open', $4, $5),
		       ($1, $2, null, 'Owner-only commitment', 'owner-private-description', '2026-07-01T16:00:00Z', 'open', $5, $5)
	`, fx.workspaceID, eventID, contactID, memberPersonID, ownerID); err != nil {
		t.Fatal(err)
	}
	insertTicketWithPaymentStatus(t, fx, eventID, "ticket-private@example.test", "Ticket Private", "paid", 1500, "usd")

	portal := getParticipantPortal(t, fx.app, fx.memberCookie, http.StatusOK)
	if got := portal.Assignments; len(got) != 1 || got[0].Title != "Soundcheck" || got[0].Kind != "shift" || got[0].StartsAt == nil || got[0].EndsAt == nil || got[0].Status != "assigned" || got[0].ParticipantRequirements != "Bring a DI and arrive 15 minutes early." {
		t.Fatalf("assignments = %#v", got)
	}
	if got := portal.Commitments; len(got) != 1 || got[0].Title != "Bring cables" || got[0].DueAt == nil || got[0].Status != "open" {
		t.Fatalf("commitments = %#v", got)
	}

	body, err := json.Marshal(portal)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateValue := range []string{
		"operator-only staffing note", "application-only private note", "application-only requirement", "application-private-message",
		"contact-private@example.test", "contact-private-note", "commitment-private-description",
		"Owner-only commitment", "ticket-private@example.test", "Ticket Private",
		"amountCents", "paymentStatus", "currency",
	} {
		if strings.Contains(string(body), privateValue) {
			t.Fatalf("portal exposed private value %q: %s", privateValue, body)
		}
	}

	if _, err := fx.app.db.Exec(t.Context(), `update event_staffing_items set assigned_person_id = $2 where id = $1`, staffingID, ownerID); err != nil {
		t.Fatal(err)
	}
	reassigned := getParticipantPortal(t, fx.app, fx.memberCookie, http.StatusOK)
	if len(reassigned.Assignments) != 0 || len(reassigned.Commitments) != 0 {
		t.Fatalf("reassigned portal = %#v, want empty", reassigned)
	}

	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at = now() where id = $1`, memberRowID); err != nil {
		t.Fatal(err)
	}
	revoked := getParticipantPortal(t, fx.app, fx.memberCookie, http.StatusOK)
	if len(revoked.Assignments) != 0 || len(revoked.Commitments) != 0 {
		t.Fatalf("revoked portal = %#v, want empty", revoked)
	}
}

func TestParticipantPortalRequiresAuthenticatedPerson(t *testing.T) {
	fx := newLifecycleFixture(t)
	getParticipantPortal(t, fx.app, nil, http.StatusUnauthorized)
}

func getParticipantPortal(t *testing.T, app *App, cookie *http.Cookie, wantStatus int) participantPortalDTO {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/me/participant-portal", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, req)
	if response.Code != wantStatus {
		t.Fatalf("participant portal status=%d body=%s, want %d", response.Code, response.Body.String(), wantStatus)
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("cache control = %q", got)
	}
	if wantStatus != http.StatusOK {
		return participantPortalDTO{}
	}
	var portal participantPortalDTO
	if err := json.Unmarshal(response.Body.Bytes(), &portal); err != nil {
		t.Fatal(err)
	}
	return portal
}

func TestParticipantShiftCalendarExportsOnlyCurrentPersonTimedAssignments(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Calendar Export Event", 10), "id")
	memberPersonID, memberRowID := memberIdentity(t, fx)
	ownerID := ownerPersonID(t, fx)

	insertItem := func(title, status string, personID any, startsAt, endsAt any) string {
		t.Helper()
		var id string
		if err := fx.app.db.QueryRow(t.Context(), `
			insert into event_staffing_items (event_id, title, kind, notes, participant_requirements, starts_at, ends_at, assigned_person_id, status, created_by_person_id)
			values ($1, $2, 'shift', 'operator-only staffing note', 'Bring earplugs; arrive early.', $3, $4, $5, $6, $7)
			returning id
		`, eventID, title, startsAt, endsAt, personID, status, ownerID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	shiftID := insertItem("Door shift", "assigned", memberPersonID, "2026-07-01T18:00:00Z", "2026-07-01T19:00:00Z")
	cancelledID := insertItem("Cancelled shift", "cancelled", memberPersonID, "2026-07-01T20:00:00Z", "2026-07-01T21:00:00Z")
	untimedID := insertItem("Untimed task", "assigned", memberPersonID, nil, nil)
	ownerShiftID := insertItem("Owner-only shift", "assigned", ownerID, "2026-07-01T18:00:00Z", "2026-07-01T19:00:00Z")
	insertTicketWithPaymentStatus(t, fx, eventID, "ticket-private@example.test", "Ticket Private", "paid", 1500, "usd")

	body := getParticipantShiftCalendar(t, fx.app, fx.memberCookie, http.StatusOK)
	events := parseICalendar(t, body)
	if len(events) != 1 {
		t.Fatalf("calendar events = %#v, want only the timed member shift", events)
	}
	shiftUID := "staffing-" + shiftID + "@os.subcult.tv"
	if events[0]["UID"] != shiftUID || events[0]["DTSTART"] != "20260701T180000Z" || events[0]["DTEND"] != "20260701T190000Z" {
		t.Fatalf("calendar event = %#v", events[0])
	}
	if !strings.Contains(events[0]["DESCRIPTION"], "Bring earplugs; arrive early.") {
		t.Fatalf("description lacks participant requirements: %q", events[0]["DESCRIPTION"])
	}
	for _, privateValue := range []string{
		"operator-only staffing note", cancelledID, untimedID, ownerShiftID, "Owner-only shift",
		"ticket-private@example.test", "Ticket Private", fx.email("owner"), fx.email("member"),
	} {
		if strings.Contains(body, privateValue) {
			t.Fatalf("calendar exposed private value %q: %s", privateValue, body)
		}
	}

	// Reschedule through the same columns the staffing update route writes.
	if _, err := fx.app.db.Exec(t.Context(), `update event_staffing_items set updated_at = now() - interval '1 hour' where id = $1`, shiftID); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update events set updated_at = now() - interval '1 hour' where id = $1`, eventID); err != nil {
		t.Fatal(err)
	}
	before := parseICalendar(t, getParticipantShiftCalendar(t, fx.app, fx.memberCookie, http.StatusOK))[0]
	if _, err := fx.app.db.Exec(t.Context(), `
		update event_staffing_items
		set starts_at = '2026-07-01T20:00:00Z', ends_at = '2026-07-01T22:00:00Z', updated_at = now()
		where id = $1
	`, shiftID); err != nil {
		t.Fatal(err)
	}
	after := parseICalendar(t, getParticipantShiftCalendar(t, fx.app, fx.memberCookie, http.StatusOK))[0]
	beforeSequence, _ := strconv.Atoi(before["SEQUENCE"])
	afterSequence, _ := strconv.Atoi(after["SEQUENCE"])
	if after["UID"] != shiftUID || after["DTSTART"] != "20260701T200000Z" || after["DTEND"] != "20260701T220000Z" ||
		afterSequence <= beforeSequence || after["LAST-MODIFIED"] <= before["LAST-MODIFIED"] {
		t.Fatalf("reschedule before=%#v after=%#v", before, after)
	}

	if _, err := fx.app.db.Exec(t.Context(), `update event_staffing_items set status = 'cancelled' where id = $1`, shiftID); err != nil {
		t.Fatal(err)
	}
	if cancelled := parseICalendar(t, getParticipantShiftCalendar(t, fx.app, fx.memberCookie, http.StatusOK)); len(cancelled) != 0 {
		t.Fatalf("cancelled calendar = %#v, want empty", cancelled)
	}

	if _, err := fx.app.db.Exec(t.Context(), `update event_staffing_items set status = 'assigned' where id = $1`, shiftID); err != nil {
		t.Fatal(err)
	}
	ownerEvents := parseICalendar(t, getParticipantShiftCalendar(t, fx.app, fx.ownerCookie, http.StatusOK))
	if len(ownerEvents) != 1 || ownerEvents[0]["UID"] != "staffing-"+ownerShiftID+"@os.subcult.tv" {
		t.Fatalf("owner calendar = %#v, want only the owner's shift", ownerEvents)
	}

	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at = now() where id = $1`, memberRowID); err != nil {
		t.Fatal(err)
	}
	if revoked := parseICalendar(t, getParticipantShiftCalendar(t, fx.app, fx.memberCookie, http.StatusOK)); len(revoked) != 0 {
		t.Fatalf("revoked calendar = %#v, want empty", revoked)
	}
}

func TestParticipantShiftCalendarRequiresAuthenticatedPerson(t *testing.T) {
	fx := newLifecycleFixture(t)
	getParticipantShiftCalendar(t, fx.app, nil, http.StatusUnauthorized)
}

func getParticipantShiftCalendar(t *testing.T, app *App, cookie *http.Cookie, wantStatus int) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/me/participant-portal/shifts.ics", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, req)
	if response.Code != wantStatus {
		t.Fatalf("shift calendar status=%d body=%s, want %d", response.Code, response.Body.String(), wantStatus)
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("cache control = %q", got)
	}
	if wantStatus != http.StatusOK {
		if strings.Contains(response.Body.String(), "BEGIN:VCALENDAR") {
			t.Fatalf("denied response contains calendar: %s", response.Body.String())
		}
		return ""
	}
	if got := response.Header().Get("Content-Type"); got != "text/calendar; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if got := response.Header().Get("Content-Disposition"); got != `attachment; filename="subcult-os-shifts.ics"` {
		t.Fatalf("content disposition = %q", got)
	}
	return response.Body.String()
}
