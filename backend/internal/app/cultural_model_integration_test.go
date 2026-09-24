package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// lexiconContractDirForTest resolves the admitted Lexicon contract directory
// the same way backend/internal/atproto's own tests do: relative to this
// package directory, three levels up to the repository root. Both packages
// sit at the same depth (backend/internal/<pkg>), so the same relative path
// works from either one.
const lexiconContractDirForTest = "../../../contracts/lexicons"

// field reads one key out of a decoded JSON response body as a plain object
// field lookup helper, since testResponse.JSON is decoded as `any`.
func field(t *testing.T, value any, key string) any {
	t.Helper()
	return mustObject(t, value)[key]
}

func newCulturalFixture(t *testing.T, shared ...*App) lifecycleFixture {
	t.Helper()
	if len(shared) > 0 {
		return newLifecycleFixture(t, shared[0])
	}
	db := newMigrationTestPool(t)
	if err := RunMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	app := New(Config{
		AppEnv:             "test",
		PublicWebURL:       "http://public.test",
		SessionSecret:      "test-secret",
		LexiconContractDir: lexiconContractDirForTest,
	}, db)
	if app.lexiconErr != nil {
		t.Fatalf("lexicon catalog failed to load: %v", app.lexiconErr)
	}
	return newLifecycleFixture(t, app)
}

// --- Fresh and upgrade migration paths (criterion 4) -----------------------

func TestCulturalModelFreshMigrationCreatesTables(t *testing.T) {
	ctx := t.Context()
	db := newMigrationTestPool(t)
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{
		"cultural_profiles",
		"cultural_places",
		"cultural_place_protected_details",
		"event_occurrences",
		"event_occurrence_profiles",
	} {
		var exists bool
		if err := db.QueryRow(ctx, `
			select exists (
				select 1 from information_schema.tables
				where table_schema = current_schema() and table_name = $1
			)
		`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("expected table %q to exist after fresh RunMigrations", table)
		}
	}

	version, err := CurrentSchemaVersion(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	// Every slice that adds a migration bumps minimumSchemaVersion, so a
	// fresh migration run must land exactly there.
	if version != minimumSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, minimumSchemaVersion)
	}
}

func TestCulturalModelUpgradePathPreservesExistingEvent(t *testing.T) {
	ctx := t.Context()
	db := newMigrationTestPool(t)

	migrations, err := loadMigrations(migrationFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) < 8 {
		t.Fatalf("expected at least 8 migrations, got %d", len(migrations))
	}
	// Build a genuine version-seven fixture without weakening the current
	// binary's minimum-version guard (runMigrations refuses to leave a
	// database below minimumSchemaVersion, even mid-test), following the
	// same manual-ledger approach as TestEmailMigrationHoldsLegacyAndDisabledMessages.
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `create table schema_migrations(version integer primary key,name text not null,checksum char(64) not null,applied_at timestamptz not null default now())`); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations[:7] {
		if _, err := tx.Exec(ctx, m.SQL); err != nil {
			t.Fatalf("apply migration %d: %v", m.Version, err)
		}
		if _, err := tx.Exec(ctx, `insert into schema_migrations(version,name,checksum) values($1,$2,$3)`, m.Version, m.Name, m.Checksum); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// Seed a workspace and a private operator event on the pre-upgrade schema.
	var personID, workspaceID, eventID string
	if err := db.QueryRow(ctx, `
		insert into people (email, display_name, password_hash)
		values ('seed-owner@example.test', 'Seed Owner', '')
		returning id
	`).Scan(&personID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `
		insert into workspaces (name) values ('Seed Workspace') returning id
	`).Scan(&workspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `
		insert into workspace_members (workspace_id, person_id, role) values ($1, $2, 'owner')
	`, workspaceID, personID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `
		insert into events (workspace_id, title, starts_at, public_description, location_display, ticket_allocation, created_by_person_id)
		values ($1, 'Pre-upgrade Show', now() + interval '30 days', 'A show that predates the cultural model.', 'The Hall', 50, $2)
		returning id
	`, workspaceID, personID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}

	// Apply the rest, including migration 8.
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("apply remaining migrations: %v", err)
	}

	var title string
	if err := db.QueryRow(ctx, `select title from events where id = $1`, eventID).Scan(&title); err != nil {
		t.Fatalf("pre-upgrade event did not survive migration: %v", err)
	}
	if title != "Pre-upgrade Show" {
		t.Fatalf("title = %q, want unchanged", title)
	}

	// The pre-existing event can now be related to a new public occurrence.
	var occurrenceID string
	if err := db.QueryRow(ctx, `
		insert into event_occurrences (workspace_id, event_id, name, starts_at, created_by_person_id)
		values ($1, $2, 'Pre-upgrade Show (public)', now() + interval '30 days', $3)
		returning id
	`, workspaceID, eventID, personID).Scan(&occurrenceID); err != nil {
		t.Fatalf("could not relate a new occurrence to the pre-existing event: %v", err)
	}
	if occurrenceID == "" {
		t.Fatal("expected an occurrence id")
	}
}

// --- Profile and place CRUD -------------------------------------------------

func TestCulturalProfileCRUD(t *testing.T) {
	fx := newCulturalFixture(t)

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/profiles", map[string]any{
		"kind":        "act",
		"displayName": "DJ Signal",
		"description": "A touring act.",
	}, http.StatusOK)
	profileID := mustString(t, created.JSON, "id")
	if kind := field(t, created.JSON, "kind"); kind != "act" {
		t.Fatalf("kind = %v, want act", kind)
	}

	updated := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/profiles/"+profileID, map[string]any{
		"displayName": "DJ Signal Collective",
	}, http.StatusOK)
	if name := field(t, updated.JSON, "displayName"); name != "DJ Signal Collective" {
		t.Fatalf("displayName = %v, want updated value", name)
	}
}

func TestCulturalPlaceCRUDSeparatesPublicAndProtectedFields(t *testing.T) {
	fx := newCulturalFixture(t)

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places", map[string]any{
		"name":              "The Hollow",
		"locality":          "Chicago",
		"region":            "IL",
		"country":           "US",
		"coordinatesPublic": true,
		"publicLatitude":    "41.8781",
		"publicLongitude":   "-87.6298",
		"streetAddress":     "123 Confidential Ave, Suite 4",
		"accessNotes":       "Door code 4471, ask for Sam",
	}, http.StatusOK)
	placeID := mustString(t, created.JSON, "id")
	if addr := field(t, created.JSON, "streetAddress"); addr != "123 Confidential Ave, Suite 4" {
		t.Fatalf("expected the operator-facing DTO to include the protected street address, got %v", addr)
	}

	updated := patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places/"+placeID, map[string]any{
		"locality": "Chicagoland",
	}, http.StatusOK)
	if locality := field(t, updated.JSON, "locality"); locality != "Chicagoland" {
		t.Fatalf("locality = %v, want updated value", locality)
	}
	// Protected fields are untouched by an unrelated public-field update.
	if addr := field(t, updated.JSON, "streetAddress"); addr != "123 Confidential Ave, Suite 4" {
		t.Fatalf("expected street address to survive an unrelated update, got %v", addr)
	}
}

// --- Duplicate occurrences and multi-host attribution (criterion 3) -------

func createOccurrenceProfileAndPlace(t *testing.T, fx lifecycleFixture) (profileID, placeID string) {
	t.Helper()
	profile := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/profiles", map[string]any{
		"kind":        "collective",
		"displayName": "Night Collective",
	}, http.StatusOK)
	place := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places", map[string]any{
		"name":     "Warehouse 12",
		"locality": "Chicago",
	}, http.StatusOK)
	return mustString(t, profile.JSON, "id"), mustString(t, place.JSON, "id")
}

func TestEventOccurrenceDuplicateListingsAndMultiHostCredits(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Cross-listed Show", 100)
	eventID := mustString(t, event, "id")
	profileID, placeID := createOccurrenceProfileAndPlace(t, fx)

	secondProfile := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/profiles", map[string]any{
		"kind":        "act",
		"displayName": "Guest Performer",
	}, http.StatusOK)
	secondProfileID := mustString(t, secondProfile.JSON, "id")

	occurrenceA := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"placeId":  placeID,
		"name":     "Cross-listed Show (Collective listing)",
		"startsAt": "2026-07-01T20:00:00-05:00",
		"timezone": "America/Chicago",
	}, http.StatusOK)
	occurrenceB := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"placeId":  placeID,
		"name":     "Cross-listed Show (Second listing)",
		"startsAt": "2026-07-01T20:00:00-05:00",
		"timezone": "America/Chicago",
	}, http.StatusOK)
	if field(t, occurrenceA.JSON, "id") == field(t, occurrenceB.JSON, "id") {
		t.Fatal("expected two distinct occurrence rows for the same operator event")
	}

	occurrenceAID := mustString(t, occurrenceA.JSON, "id")

	list := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", http.StatusOK)
	items, ok := list.JSON.([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected two occurrences listed for the same event, got %#v", list.JSON)
	}

	attachA := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceAID+"/credits", map[string]any{
		"profileId": profileID,
		"role":      "host",
		"sortOrder": 0,
	}, http.StatusOK)
	credits, ok := field(t, attachA.JSON, "credits").([]any)
	if !ok || len(credits) != 1 {
		t.Fatalf("expected one credit after first attach, got %#v", field(t, attachA.JSON, "credits"))
	}

	attachB := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceAID+"/credits", map[string]any{
		"profileId": secondProfileID,
		"role":      "performer",
		"sortOrder": 1,
	}, http.StatusOK)
	credits, ok = field(t, attachB.JSON, "credits").([]any)
	if !ok || len(credits) != 2 {
		t.Fatalf("expected two credits after second attach (multi-host), got %#v", field(t, attachB.JSON, "credits"))
	}

	detachPath := "/api/events/" + eventID + "/occurrences/" + occurrenceAID + "/credits/" + secondProfileID
	detached := doJSON(t, http.MethodDelete, fx.app, fx.ownerCookie, detachPath, nil, http.StatusOK)
	credits, ok = field(t, detached.JSON, "credits").([]any)
	if !ok || len(credits) != 1 {
		t.Fatalf("expected one credit after detach, got %#v", field(t, detached.JSON, "credits"))
	}
}

// --- Reschedule must not touch tickets or reservations (criterion 3) ------

func TestEventOccurrenceRescheduleDoesNotChangeTickets(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Reschedule Show", 20)
	eventID := mustString(t, event, "id")
	_, placeID := createOccurrenceProfileAndPlace(t, fx)

	ticket := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/test-ticket", map[string]any{
		"email":       "guest@example.test",
		"displayName": "Guest",
	}, http.StatusOK)
	ticketID := mustString(t, ticket.JSON, "id")
	ticketCode := mustString(t, ticket.JSON, "code")

	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"placeId":  placeID,
		"name":     "Reschedule Show (public)",
		"startsAt": "2026-08-01T20:00:00-05:00",
		"timezone": "America/Chicago",
	}, http.StatusOK)
	occurrenceID := mustString(t, occurrence.JSON, "id")

	rescheduled := patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID, map[string]any{
		"startsAt": "2026-08-08T20:00:00-05:00",
	}, http.StatusOK)
	if status := field(t, rescheduled.JSON, "status"); status != occurrenceStatusRescheduled {
		t.Fatalf("status = %v, want rescheduled", status)
	}
	if field(t, rescheduled.JSON, "startsAt") == field(t, occurrence.JSON, "startsAt") {
		t.Fatal("expected startsAt to change after reschedule")
	}

	// The private ticket row is untouched: same id, code and status.
	var status, code string
	var eventIDAfter string
	if err := fx.app.db.QueryRow(t.Context(), `
		select id, event_id, status, code from tickets where id = $1
	`, ticketID).Scan(&ticketID, &eventIDAfter, &status, &code); err != nil {
		t.Fatal(err)
	}
	if eventIDAfter != eventID || status != "reserved" || code != ticketCode {
		t.Fatalf("ticket row changed after occurrence reschedule: eventId=%q status=%q code=%q", eventIDAfter, status, code)
	}
}

// --- Cross-workspace ownership rejection (criterion 3 acceptance) ---------

func TestCulturalModelRejectsCrossWorkspaceAccess(t *testing.T) {
	fxA := newCulturalFixture(t)
	fxB := newCulturalFixture(t, fxA.app)

	event := createEvent(t, fxA, "Workspace A Show", 10)
	eventID := mustString(t, event, "id")
	profileID, placeID := createOccurrenceProfileAndPlace(t, fxA)
	occurrence := postJSON(t, fxA.app, fxA.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"placeId":  placeID,
		"name":     "Workspace A Show (public)",
		"startsAt": "2026-09-01T20:00:00Z",
	}, http.StatusOK)
	occurrenceID := mustString(t, occurrence.JSON, "id")

	// Workspace B's owner has valid UUIDs for workspace A's resources but no
	// membership in workspace A; every read/write must be rejected.
	getJSON(t, fxB.app, fxB.ownerCookie, "/api/events/"+eventID+"/occurrences", http.StatusForbidden)
	patchJSON(t, fxB.app, fxB.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID, map[string]any{
		"name": "Hijacked",
	}, http.StatusForbidden)

	// Workspace B's own profile cannot be credited on workspace A's occurrence.
	bProfile := postJSON(t, fxB.app, fxB.ownerCookie, "/api/workspaces/"+fxB.workspaceID+"/profiles", map[string]any{
		"displayName": "Workspace B Act",
	}, http.StatusOK)
	bProfileID := mustString(t, bProfile.JSON, "id")
	postJSON(t, fxA.app, fxA.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID+"/credits", map[string]any{
		"profileId": bProfileID,
	}, http.StatusBadRequest)

	// A place from workspace B cannot be attached to workspace A's occurrence.
	bPlace := postJSON(t, fxB.app, fxB.ownerCookie, "/api/workspaces/"+fxB.workspaceID+"/places", map[string]any{
		"name": "Workspace B Venue",
	}, http.StatusOK)
	bPlaceID := mustString(t, bPlace.JSON, "id")
	postJSON(t, fxA.app, fxA.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"placeId":  bPlaceID,
		"name":     "Should fail",
		"startsAt": "2026-09-02T20:00:00Z",
	}, http.StatusBadRequest)

	_ = profileID
}

// --- DST correctness (criterion 2) -----------------------------------------

func TestEventOccurrenceDSTChicagoRoundTrip(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Spring Forward Show", 10)
	eventID := mustString(t, event, "id")

	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	// 2026-03-08 is the US spring-forward date: 2:00am local jumps to 3:00am.
	// Storing an unambiguous wall time before the jump and one after it, and
	// reading both back through the stored timestamptz + timezone column,
	// proves DST is handled via absolute instants rather than naive wall-time
	// arithmetic (which would either double-count or lose the missing hour).
	starts := time.Date(2026, 3, 7, 23, 0, 0, 0, chicago)
	ends := time.Date(2026, 3, 8, 4, 0, 0, 0, chicago)

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"name":     "Spring Forward Show (public)",
		"startsAt": starts.Format(time.RFC3339),
		"endsAt":   ends.Format(time.RFC3339),
		"timezone": "America/Chicago",
	}, http.StatusOK)

	gotStartsAt, err := time.Parse(time.RFC3339Nano, mustString(t, created.JSON, "startsAt"))
	if err != nil {
		t.Fatal(err)
	}
	gotEndsAt, err := time.Parse(time.RFC3339Nano, mustString(t, created.JSON, "endsAt"))
	if err != nil {
		t.Fatal(err)
	}

	if !gotStartsAt.Equal(starts) {
		t.Fatalf("startsAt instant = %s, want %s", gotStartsAt, starts)
	}
	if !gotEndsAt.Equal(ends) {
		t.Fatalf("endsAt instant = %s, want %s", gotEndsAt, ends)
	}

	// Reading the stored instant back in the stored timezone must reproduce
	// the original local wall-clock hours exactly, and the wall-clock gap
	// (5h) must differ from the real elapsed instant gap (4h, because the
	// clock skipped 02:00-03:00).
	localStart := gotStartsAt.In(chicago)
	localEnd := gotEndsAt.In(chicago)
	if localStart.Hour() != 23 || localEnd.Hour() != 4 {
		t.Fatalf("local wall clock round-trip = %02d:00 .. %02d:00, want 23:00 .. 04:00", localStart.Hour(), localEnd.Hour())
	}
	if localStart.Format("MST") != "CST" {
		t.Fatalf("startsAt zone abbreviation = %s, want CST (pre-DST)", localStart.Format("MST"))
	}
	if localEnd.Format("MST") != "CDT" {
		t.Fatalf("endsAt zone abbreviation = %s, want CDT (post-DST)", localEnd.Format("MST"))
	}
	if gotEndsAt.Sub(gotStartsAt) != 4*time.Hour {
		t.Fatalf("elapsed instant duration = %s, want 4h (one wall-clock hour lost to spring-forward)", gotEndsAt.Sub(gotStartsAt))
	}
}

// --- Public serializer sentinels (criterion 4) ------------------------------

func TestOccurrencePublicPreviewNeverLeaksProtectedOrOperationalData(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEventWithPricing(t, fx, "Ticketed Show", 10, "fixed", 4200, "usd")
	eventID := mustString(t, event, "id")

	const streetAddressSentinel = "999 Confidential Ln, Do Not Publish"
	const accessNotesSentinel = "Backstage code 7-7-3-1, ask for the private notes"
	const staffingSentinel = "Private staffing note: door lead is Alex, contact 555-0100"
	const ticketBuyerSentinel = "secret-buyer@example.test"

	place := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places", map[string]any{
		"name":          "Sentinel Venue",
		"locality":      "Chicago",
		"region":        "IL",
		"country":       "US",
		"streetAddress": streetAddressSentinel,
		"accessNotes":   accessNotesSentinel,
	}, http.StatusOK)
	placeID := mustString(t, place.JSON, "id")

	profile := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/profiles", map[string]any{
		"kind":        "collective",
		"displayName": "Public Collective",
		"description": "Publicly fine description.",
	}, http.StatusOK)
	profileID := mustString(t, profile.JSON, "id")

	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"placeId":  placeID,
		"name":     "Ticketed Show (public)",
		"startsAt": "2026-10-01T20:00:00-05:00",
		"timezone": "America/Chicago",
	}, http.StatusOK)
	occurrenceID := mustString(t, occurrence.JSON, "id")

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID+"/credits", map[string]any{
		"profileId": profileID,
	}, http.StatusOK)

	// Seed a ticket with a price and a staffing row with private notes; these
	// tables are never read by the public projection, but we prove it here
	// rather than merely asserting by code inspection.
	if _, err := fx.app.db.Exec(t.Context(), `
		insert into tickets (event_id, email, display_name, code, status, payment_status, amount_cents, currency)
		values ($1, $2, 'Secret Buyer', 'SENTINEL1', 'reserved', 'paid', 4200, 'usd')
	`, eventID, ticketBuyerSentinel); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/staffing", map[string]any{
		"title": "Door",
		"kind":  "shift",
		"notes": staffingSentinel,
	}, http.StatusOK)

	preview := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID+"/public-preview", http.StatusOK)
	rawPreview, err := json.Marshal(preview.JSON)
	if err != nil {
		t.Fatal(err)
	}
	body := string(rawPreview)

	for _, sentinel := range []string{streetAddressSentinel, accessNotesSentinel, staffingSentinel, ticketBuyerSentinel, "4200"} {
		if strings.Contains(body, sentinel) {
			t.Fatalf("public preview leaked sentinel %q: %s", sentinel, body)
		}
	}

	// The preview must still carry the legitimately public fields.
	for _, want := range []string{"Sentinel Venue", "Public Collective", "Ticketed Show (public)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("public preview missing expected public field %q: %s", want, body)
		}
	}
}

func TestOccurrencePublicPreviewRequiresACreditedProfile(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "No Credit Show", 10)
	eventID := mustString(t, event, "id")

	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"name":     "No Credit Show (public)",
		"startsAt": "2026-11-01T20:00:00Z",
	}, http.StatusOK)
	occurrenceID := mustString(t, occurrence.JSON, "id")

	getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID+"/public-preview", http.StatusBadRequest)
}
