package app

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// privacySentinelFixture (SEC-15 / Issue #15) plants a unique, greppable
// sentinel value in every private column this repository stores across
// contacts, staffing, role applications, ticket/attendance rows and the
// MODEL-01 protected place details, then exposes everything a test needs to
// assert none of them ever appear in an anonymous response, the public
// preview, or captured log output. This is the reusable "private-field
// sentinel" helper the privacy audit
// (docs/development/privacy-audit-2026-09-23.md) calls for; extend it here
// rather than hand-rolling a parallel fixture when new private tables are
// added.
type privacySentinelFixture struct {
	fx lifecycleFixture

	eventID string
	slug    string

	contactID    string
	contactEmail string
	contactNotes string

	roleID           string
	applicationEmail string
	applicationMsg   string

	ticketEmail       string
	ticketDisplayName string
	ticketCode        string

	staffingNotes string

	placeID           string
	placeStreetSecret string
	placeAccessSecret string

	occurrenceID string
	profileID    string
}

// sentinels returns every planted private value in one slice, for a single
// leak-scan pass over an arbitrary response or log body.
func (f privacySentinelFixture) sentinels() []string {
	return []string{
		f.contactEmail,
		f.contactNotes,
		f.applicationEmail,
		f.applicationMsg,
		f.ticketEmail,
		f.ticketDisplayName,
		f.placeStreetSecret,
		f.placeAccessSecret,
		f.staffingNotes,
	}
}

// assertNoSentinelLeak fails the test if any planted private sentinel value
// appears in body, labeling the failure with which route/body produced it.
func assertNoSentinelLeak(t *testing.T, label, body string, sentinels []string) {
	t.Helper()
	for _, sentinel := range sentinels {
		if sentinel == "" {
			continue
		}
		if strings.Contains(body, sentinel) {
			t.Fatalf("%s leaked private sentinel %q: %s", label, sentinel, body)
		}
	}
}

// plantPrivacySentinels builds one workspace with one published, ticketed,
// role-accepting event and seeds a unique sentinel value into every private
// column class named in Issue #15: contacts, staffing notes, a role
// applicant's email/message, a ticket holder's email/display name
// (reservation/attendance), and a cultural place's protected street
// address/access notes (MODEL-01). It credits a cultural profile to the
// event occurrence so the public preview (criterion 4 / MODEL-01) is
// reachable from the same fixture.
func plantPrivacySentinels(t *testing.T, fx lifecycleFixture) privacySentinelFixture {
	t.Helper()

	out := privacySentinelFixture{fx: fx}

	event := createEventWithPricing(t, fx, "Sentinel Benefit Show", 20, "fixed", 2500, "usd")
	out.eventID = mustString(t, event, "id")
	published := publishEvent(t, fx, out.eventID)
	out.slug = mustString(t, published, "publicSlug")

	out.contactEmail = "sentinel-contact+" + fx.suffix + "@example.test"
	out.contactNotes = "SENTINEL contact notes " + fx.suffix
	contact := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/contacts", map[string]any{
		"displayName": "Sentinel Contact",
		"email":       out.contactEmail,
		"phone":       "+15555550199",
		"notes":       out.contactNotes,
		"tags":        []string{"sentinel"},
	}, http.StatusOK)
	out.contactID = mustString(t, contact.JSON, "id")

	role := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+out.eventID+"/roles", map[string]any{
		"name":        "Sentinel Role",
		"description": "Sentinel role description",
		"capacity":    3,
		"public":      true,
	}, http.StatusOK)
	out.roleID = mustString(t, role.JSON, "id")

	out.applicationEmail = "sentinel-applicant+" + fx.suffix + "@example.test"
	out.applicationMsg = "SENTINEL applicant message " + fx.suffix
	postJSON(t, fx.app, nil, "/api/public/events/"+out.slug+"/role-applications", map[string]any{
		"roleId":         out.roleID,
		"applicantName":  "Sentinel Applicant",
		"applicantEmail": out.applicationEmail,
		"message":        out.applicationMsg,
	}, http.StatusOK)

	out.ticketEmail = "sentinel-ticket+" + fx.suffix + "@example.test"
	out.ticketDisplayName = "Sentinel Ticket Holder " + fx.suffix
	out.ticketCode = insertTicketWithPaymentStatus(t, fx, out.eventID, out.ticketEmail, out.ticketDisplayName, "paid", 2500, "usd")
	// insertTicketWithPaymentStatus returns the ticket id, not the code; load
	// the actual capability code for the anonymous /api/tickets/{code} route.
	var code string
	if err := fx.app.db.QueryRow(t.Context(), `select code from tickets where id = $1`, out.ticketCode).Scan(&code); err != nil {
		t.Fatal(err)
	}
	out.ticketCode = code

	out.staffingNotes = "SENTINEL staffing note " + fx.suffix
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+out.eventID+"/staffing", map[string]any{
		"title": "Sentinel Door Shift",
		"kind":  "shift",
		"notes": out.staffingNotes,
	}, http.StatusOK)

	out.placeStreetSecret = "SENTINEL street address " + fx.suffix
	out.placeAccessSecret = "SENTINEL access notes " + fx.suffix
	place := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places", map[string]any{
		"name":          "Sentinel Venue",
		"locality":      "Chicago",
		"region":        "IL",
		"country":       "US",
		"streetAddress": out.placeStreetSecret,
		"accessNotes":   out.placeAccessSecret,
	}, http.StatusOK)
	out.placeID = mustString(t, place.JSON, "id")

	profile := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/profiles", map[string]any{
		"kind":        "collective",
		"displayName": "Sentinel Collective",
		"description": "Publicly fine description.",
	}, http.StatusOK)
	out.profileID = mustString(t, profile.JSON, "id")

	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+out.eventID+"/occurrences", map[string]any{
		"placeId":  out.placeID,
		"name":     "Sentinel Benefit Show (public)",
		"startsAt": "2026-10-01T20:00:00-05:00",
		"timezone": "America/Chicago",
	}, http.StatusOK)
	out.occurrenceID = mustString(t, occurrence.JSON, "id")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+out.eventID+"/occurrences/"+out.occurrenceID+"/credits", map[string]any{
		"profileId": out.profileID,
	}, http.StatusOK)

	return out
}

// TestPrivacySentinelsNeverLeakIntoAnonymousRoutes exercises Issue #15
// criterion (b): plant a sentinel in every private column class and prove
// none of them appear in any anonymous response, including the MODEL-01
// public preview (criterion (c) is covered separately below).
func TestPrivacySentinelsNeverLeakIntoAnonymousRoutes(t *testing.T) {
	fx := newCulturalFixture(t)
	sentinels := plantPrivacySentinels(t, fx)
	all := sentinels.sentinels()

	anonymous := map[string]testResponse{
		"public discovery":      getJSON(t, fx.app, nil, "/api/public/events", http.StatusOK),
		"public event detail":   getJSON(t, fx.app, nil, "/api/public/events/"+sentinels.slug, http.StatusOK),
		"public event roles":    getJSON(t, fx.app, nil, "/api/public/events/"+sentinels.slug+"/roles", http.StatusOK),
		"ticket capability url": getJSON(t, fx.app, nil, "/api/tickets/"+sentinels.ticketCode, http.StatusOK),
	}
	for label, resp := range anonymous {
		// The ticket capability route legitimately echoes the holder's own
		// email/display name back to whoever presents the unguessable code
		// (see privacy-audit-2026-09-23.md); every other private sentinel
		// must still be absent.
		forbidden := make([]string, 0, len(all))
		for _, s := range all {
			if label == "ticket capability url" && (s == sentinels.ticketEmail || s == sentinels.ticketDisplayName) {
				continue
			}
			forbidden = append(forbidden, s)
		}
		assertNoSentinelLeak(t, label, resp.Body, forbidden)
	}

	// The public preview must never carry any private sentinel, including
	// the protected place details (MODEL-01 / criterion c).
	preview := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+sentinels.eventID+"/occurrences/"+sentinels.occurrenceID+"/public-preview", http.StatusOK)
	assertNoSentinelLeak(t, "occurrence public preview", preview.Body, all)
}

// TestCulturalPlaceProtectedDetailsRequireWorkspaceMembership covers Issue
// #15 criterion (c): cultural_place_protected_details (street address,
// access notes) may only be read by an authenticated member of the owning
// workspace, never anonymously and never by a member of a different
// workspace (no viewer/lower-privilege role exists yet, so "owner" and
// "member" are the full set of appropriate roles today).
func TestCulturalPlaceProtectedDetailsRequireWorkspaceMembership(t *testing.T) {
	fx := newCulturalFixture(t)
	sentinels := plantPrivacySentinels(t, fx)

	// A person authenticated in a different workspace must not be able to
	// read or write the protected details by guessing the place ID.
	other := newCulturalFixture(t, fx.app)
	forbiddenList := getJSON(t, fx.app, other.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places", http.StatusForbidden)
	assertNoSentinelLeak(t, "cross-workspace place list", forbiddenList.Body, []string{sentinels.placeStreetSecret, sentinels.placeAccessSecret})

	forbiddenUpdate := patchJSON(t, fx.app, other.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places/"+sentinels.placeID, map[string]any{
		"accessNotes": "attempted overwrite",
	}, http.StatusForbidden)
	assertNoSentinelLeak(t, "cross-workspace place update", forbiddenUpdate.Body, []string{sentinels.placeStreetSecret, sentinels.placeAccessSecret})

	// No anonymous route reaches cultural_places or
	// cultural_place_protected_details at all; the operator-only list route
	// is unauthenticated-rejected outright.
	unauth := getJSON(t, fx.app, nil, "/api/workspaces/"+fx.workspaceID+"/places", http.StatusUnauthorized)
	assertNoSentinelLeak(t, "anonymous place list", unauth.Body, []string{sentinels.placeStreetSecret, sentinels.placeAccessSecret})

	// A legitimate member of the owning workspace continues to read the
	// protected details over the operator API (this is the "appropriate
	// role" the criterion requires, not an absence of any access).
	allowedList := getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/places", http.StatusOK)
	if !strings.Contains(allowedList.Body, sentinels.placeStreetSecret) || !strings.Contains(allowedList.Body, sentinels.placeAccessSecret) {
		t.Fatalf("expected workspace member to read protected place details: %s", allowedList.Body)
	}
}

// TestRequestLoggerNeverRecordsQueryOrBody covers the audit requirement that
// the log middleware never records query strings or bodies containing email
// addresses (or any other user-supplied value): it must log only the
// server-owned route template, method, status and duration.
func TestRequestLoggerNeverRecordsQueryOrBody(t *testing.T) {
	app := &App{config: Config{PublicWebURL: "https://events.example.test"}, mux: http.NewServeMux()}
	const sentinelEmail = "sentinel-log-leak@example.test"
	app.mux.HandleFunc("POST /api/test/log-sentinel", func(w http.ResponseWriter, r *http.Request) {
		// Deliberately read the query/body so a naive logger that captured
		// r.URL.RawQuery or the body would have the sentinel available.
		_ = r.URL.Query().Get("email")
		buf := make([]byte, 4096)
		_, _ = r.Body.Read(buf)
		w.WriteHeader(http.StatusOK)
	})

	var captured bytes.Buffer
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	log.SetOutput(&captured)
	defer func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
	}()

	req := httptest.NewRequest(http.MethodPost, "/api/test/log-sentinel?email="+sentinelEmail, strings.NewReader(`{"email":"`+sentinelEmail+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.requestLogger(app.mux).ServeHTTP(rec, req)

	logged := captured.String()
	if strings.Contains(logged, sentinelEmail) {
		t.Fatalf("request logger recorded a query/body sentinel: %s", logged)
	}
	if strings.Contains(logged, "email=") || strings.Contains(logged, "?") {
		t.Fatalf("request logger recorded a raw query string: %s", logged)
	}
	if !strings.Contains(logged, "route=") || !strings.Contains(logged, "status=200") {
		t.Fatalf("request logger did not record the expected route/status fields: %s", logged)
	}
}
