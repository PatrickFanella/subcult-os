package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v85"
)

func TestFirstEventLifecycle(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 2)
	eventID := mustString(t, event, "id")
	published := publishEvent(t, fx, eventID)
	slug := mustString(t, published, "publicSlug")
	guestEmail := fx.email("guest")
	ticket := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": guestEmail, "displayName": "Guest"}, http.StatusOK)
	checkIn := postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": mustString(t, ticket.JSON, "code")}, http.StatusOK)
	report := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	reportObj := mustObject(t, report.JSON)

	if int(reportObj["ticketsReserved"].(float64)) != 1 || int(reportObj["ticketsCheckedIn"].(float64)) != 1 || int(reportObj["noShows"].(float64)) != 0 {
		t.Fatalf("unexpected report counts: %#v", report.JSON)
	}
	if mustString(t, checkIn.JSON, "status") != "checked_in" {
		t.Fatalf("unexpected check-in status: %#v", checkIn.JSON)
	}
}

func TestFirstEventLifecycleFreeReportSettlementSummary(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 4)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("free"), "Free Guest", "free", 0, "usd")
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("cancelled"), "Cancelled Guest", "cancelled", 1500, "usd")

	report := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	reportObj := mustObject(t, report.JSON)
	summary := mustObject(t, reportObj["settlementSummary"])

	if summary["currency"] != "usd" || int(summary["grossPaidRevenueCents"].(float64)) != 0 || int(summary["paidTicketCount"].(float64)) != 0 || int(summary["pendingTicketCount"].(float64)) != 0 || int(summary["cancelledTicketCount"].(float64)) != 1 || int(summary["freeTicketCount"].(float64)) != 1 || int(summary["reservedCount"].(float64)) != 1 {
		t.Fatalf("unexpected settlement summary: %#v", summary)
	}
}

func TestFirstEventLifecycleSettlementAPI(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 4)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("paid"), "Paid Guest", "paid", 1500, "usd")
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("free"), "Free Guest", "free", 0, "usd")
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("cancelled"), "Cancelled Guest", "cancelled", 1500, "usd")

	getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement", http.StatusNotFound)

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	ownerSettlement := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement", http.StatusOK)
	memberSettlement := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement", http.StatusOK)
	if !reflect.DeepEqual(ownerSettlement.JSON, memberSettlement.JSON) {
		t.Fatalf("expected owner/member settlement responses to match: owner=%#v member=%#v", ownerSettlement.JSON, memberSettlement.JSON)
	}

	settlement := mustObject(t, ownerSettlement.JSON)
	if settlement["eventId"] != eventID || settlement["currency"] != "usd" || int(settlement["grossPaidRevenueCents"].(float64)) != 1500 || int(settlement["paidTicketCount"].(float64)) != 1 || int(settlement["pendingTicketCount"].(float64)) != 0 || int(settlement["cancelledTicketCount"].(float64)) != 1 || int(settlement["freeTicketCount"].(float64)) != 1 || int(settlement["reservedCount"].(float64)) != 2 || int(settlement["adjustmentTotalCents"].(float64)) != 0 || int(settlement["netTotalCents"].(float64)) != 1500 || settlement["status"] != "open" {
		t.Fatalf("unexpected settlement payload: %#v", settlement)
	}
	if generatedAt, ok := settlement["generatedAt"].(string); !ok || generatedAt == "" {
		t.Fatalf("expected generatedAt timestamp, got %#v", settlement["generatedAt"])
	}
	if adjustments, ok := settlement["adjustments"].([]any); !ok || len(adjustments) != 0 {
		t.Fatalf("expected empty adjustments, got %#v", settlement["adjustments"])
	}

	getJSON(t, fx.app, nil, "/api/events/"+eventID+"/settlement", http.StatusForbidden)
	otherFx := newLifecycleFixture(t)
	getJSON(t, fx.app, otherFx.memberCookie, "/api/events/"+eventID+"/settlement", http.StatusForbidden)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{"amountCents": 100, "label": "Member"}, http.StatusForbidden)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement/finalize", map[string]any{}, http.StatusForbidden)

	firstFinalize := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/finalize", map[string]any{}, http.StatusOK)
	firstFinalized := mustObject(t, firstFinalize.JSON)
	if firstFinalized["status"] != "finalized" {
		t.Fatalf("expected finalized settlement status, got %#v", firstFinalize.JSON)
	}
	if finalizedAt, ok := firstFinalized["finalizedAt"].(string); !ok || finalizedAt == "" {
		t.Fatalf("expected finalizedAt timestamp, got %#v", firstFinalized["finalizedAt"])
	}
	if firstFinalized["finalizedByPersonId"] != ownerPersonID(t, fx) {
		t.Fatalf("expected finalizedByPersonId to match owner, got %#v", firstFinalized["finalizedByPersonId"])
	}

	secondFinalize := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/finalize", map[string]any{}, http.StatusOK)
	if !reflect.DeepEqual(firstFinalize.JSON, secondFinalize.JSON) {
		t.Fatalf("expected finalize to be idempotent: first=%#v second=%#v", firstFinalize.JSON, secondFinalize.JSON)
	}

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{"amountCents": 100, "label": "Late adjustment"}, http.StatusConflict)
}

func TestFirstEventLifecycleArchiveAPI(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 4)
	eventID := mustString(t, event, "id")

	getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive", http.StatusNotFound)

	publishEvent(t, fx, eventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	ownerArchive := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive", http.StatusOK)
	memberArchive := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/archive", http.StatusOK)
	if !reflect.DeepEqual(ownerArchive.JSON, memberArchive.JSON) {
		t.Fatalf("expected owner/member archive responses to match: owner=%#v member=%#v", ownerArchive.JSON, memberArchive.JSON)
	}

	archive := mustObject(t, ownerArchive.JSON)
	if archive["eventId"] != eventID || archive["status"] != "private" || int(archive["noteCount"].(float64)) != 0 {
		t.Fatalf("unexpected archive response: %#v", archive)
	}
	if archive["reportId"] == "" || archive["settlementId"] == "" || archive["createdAt"] == "" || archive["updatedAt"] == "" {
		t.Fatalf("archive response missing references/timestamps: %#v", archive)
	}
	notes, ok := archive["notes"].([]any)
	if !ok || len(notes) != 0 {
		t.Fatalf("expected empty notes array, got %#v", archive["notes"])
	}

	getJSON(t, fx.app, nil, "/api/events/"+eventID+"/archive", http.StatusForbidden)
	otherFx := newLifecycleFixture(t)
	getJSON(t, fx.app, otherFx.memberCookie, "/api/events/"+eventID+"/archive", http.StatusForbidden)
}

func TestEventRoleDefinitionsAPI(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 4)
	eventID := mustString(t, event, "id")
	roleSlug := "event-role-definitions-" + strings.ReplaceAll(fx.suffix, "_", "-")
	if _, err := fx.app.db.Exec(t.Context(), `
		update events
		set public_slug = $2
		where id = $1
	`, eventID, roleSlug); err != nil {
		t.Fatal(err)
	}

	if ownerList := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/roles", http.StatusOK); len(ownerList.JSON.([]any)) != 0 {
		t.Fatalf("expected empty role list, got %#v", ownerList.JSON)
	}
	if memberList := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/roles", http.StatusOK); len(memberList.JSON.([]any)) != 0 {
		t.Fatalf("expected empty member role list, got %#v", memberList.JSON)
	}

	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/roles", map[string]any{"name": "Denied", "description": "Member", "capacity": 1, "public": true}, http.StatusForbidden)

	performer := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/roles", map[string]any{"name": " Performer ", "description": " Play a 20-minute set. ", "capacity": 3, "public": true}, http.StatusOK)
	performerRole := mustObject(t, performer.JSON)
	if performerRole["eventId"] != eventID || performerRole["name"] != "Performer" || performerRole["description"] != "Play a 20-minute set." || int(performerRole["capacity"].(float64)) != 3 || performerRole["public"] != true || performerRole["active"] != true || performerRole["id"] == "" || performerRole["createdAt"] == "" || performerRole["updatedAt"] == "" {
		t.Fatalf("unexpected role response: %#v", performerRole)
	}

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/roles", map[string]any{"name": " Host ", "description": "Run the door", "capacity": 0}, http.StatusOK)
	if _, err := fx.app.db.Exec(t.Context(), `
		insert into event_roles (event_id, name, description, capacity, "public", active, created_by_person_id, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	`, eventID, "Backstage", "Private notes", 0, false, true, ownerPersonID(t, fx), time.Now().Add(2*time.Minute).UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `
		insert into event_roles (event_id, name, description, capacity, "public", active, created_by_person_id, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	`, eventID, "Disabled", "Inactive role", 0, true, false, ownerPersonID(t, fx), time.Now().Add(3*time.Minute).UTC()); err != nil {
		t.Fatal(err)
	}

	unpublished := getJSON(t, fx.app, nil, "/api/public/events/"+roleSlug+"/roles", http.StatusNotFound)
	if unpublished.Status != http.StatusNotFound {
		t.Fatalf("expected unpublished public roles to 404, got %d", unpublished.Status)
	}

	published := publishEvent(t, fx, eventID)
	if mustString(t, published, "publicSlug") != roleSlug {
		t.Fatalf("expected publish to preserve manual slug, got %#v", published)
	}

	publicRoles := getJSON(t, fx.app, nil, "/api/public/events/"+roleSlug+"/roles", http.StatusOK)
	publicRoleList := publicRoles.JSON.([]any)
	if len(publicRoleList) != 2 {
		t.Fatalf("expected two public roles, got %#v", publicRoles.JSON)
	}
	if mustObject(t, publicRoleList[0])["name"] != "Performer" || mustObject(t, publicRoleList[1])["name"] != "Host" {
		t.Fatalf("expected public roles ordered by creation, got %#v", publicRoles.JSON)
	}

	ownerRoles := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/roles", http.StatusOK).JSON.([]any)
	memberRoles := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/roles", http.StatusOK).JSON.([]any)
	if len(ownerRoles) != 4 || len(memberRoles) != 4 {
		t.Fatalf("expected owner/member to read all roles, owner=%#v member=%#v", ownerRoles, memberRoles)
	}
	if !reflect.DeepEqual(ownerRoles, memberRoles) {
		t.Fatalf("expected owner/member role reads to match, owner=%#v member=%#v", ownerRoles, memberRoles)
	}
	if mustObject(t, ownerRoles[2])["name"] != "Backstage" || mustObject(t, ownerRoles[3])["name"] != "Disabled" {
		t.Fatalf("expected private/inactive roles to be retained privately, got %#v", ownerRoles)
	}
}

func TestWorkspaceArchiveIndexAPI(t *testing.T) {
	fx := newLifecycleFixture(t)
	firstEvent := createEvent(t, fx, "Night Market", 4)
	firstEventID := mustString(t, firstEvent, "id")
	publishEvent(t, fx, firstEventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+firstEventID+"/end-of-night", map[string]any{}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+firstEventID+"/archive/notes", map[string]any{"body": "Move doors earlier."}, http.StatusOK)

	secondEvent := createEvent(t, fx, "Late Market", 4)
	secondEventID := mustString(t, secondEvent, "id")
	publishEvent(t, fx, secondEventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+secondEventID+"/end-of-night", map[string]any{}, http.StatusOK)

	seeded := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+firstEventID+"/archive/seed-draft", map[string]any{}, http.StatusOK)
	seededEventID := mustString(t, seeded.JSON, "id")

	resp := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusOK)
	archives, ok := resp.JSON.([]any)
	if !ok || len(archives) != 2 {
		t.Fatalf("expected two archive summaries, got %#v", resp.JSON)
	}

	latest := mustObject(t, archives[0])
	older := mustObject(t, archives[1])
	if latest["eventId"] != secondEventID || latest["title"] != "Late Market" || int(latest["noteCount"].(float64)) != 0 {
		t.Fatalf("unexpected latest archive summary: %#v", latest)
	}
	if latest["seededEventId"] != nil {
		t.Fatalf("did not expect seededEventId on latest archive: %#v", latest)
	}
	if latest["id"] == "" || latest["reportId"] == "" || latest["settlementId"] == "" || latest["locationDisplay"] == "" || latest["createdAt"] == "" || latest["updatedAt"] == "" {
		t.Fatalf("latest archive summary missing required fields: %#v", latest)
	}
	if older["eventId"] != firstEventID || older["title"] != "Night Market" || int(older["noteCount"].(float64)) != 1 {
		t.Fatalf("unexpected older archive summary: %#v", older)
	}
	if older["seededEventId"] != seededEventID {
		t.Fatalf("expected seeded draft id in archive summary, got %#v", older["seededEventId"])
	}

	titleSearch := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives?q=night", http.StatusOK)
	titleMatches, ok := titleSearch.JSON.([]any)
	if !ok || len(titleMatches) != 1 || mustObject(t, titleMatches[0])["eventId"] != firstEventID {
		t.Fatalf("expected title search to match first archive only, got %#v", titleSearch.JSON)
	}

	noteSearch := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives?q=doors", http.StatusOK)
	noteMatches, ok := noteSearch.JSON.([]any)
	if !ok || len(noteMatches) != 1 || mustObject(t, noteMatches[0])["eventId"] != firstEventID {
		t.Fatalf("expected note-body search to match first archive only, got %#v", noteSearch.JSON)
	}
	if _, ok := mustObject(t, noteMatches[0])["notes"]; ok {
		t.Fatalf("workspace archive summaries must not include note bodies: %#v", noteMatches[0])
	}

	missingSearch := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives?q=missing", http.StatusOK)
	if matches, ok := missingSearch.JSON.([]any); !ok || len(matches) != 0 {
		t.Fatalf("expected missing search to return empty array, got %#v", missingSearch.JSON)
	}

	percentSearch := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives?q=%25", http.StatusOK)
	if matches, ok := percentSearch.JSON.([]any); !ok || len(matches) != 0 {
		t.Fatalf("expected percent search to return empty array, got %#v", percentSearch.JSON)
	}

	underscoreSearch := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives?q=_", http.StatusOK)
	if matches, ok := underscoreSearch.JSON.([]any); !ok || len(matches) != 0 {
		t.Fatalf("expected underscore search to return empty array, got %#v", underscoreSearch.JSON)
	}

	other := newLifecycleFixture(t)
	otherEvent := createEvent(t, other, "Night Market", 4)
	otherEventID := mustString(t, otherEvent, "id")
	publishEvent(t, other, otherEventID)
	postJSON(t, other.app, other.ownerCookie, "/api/events/"+otherEventID+"/end-of-night", map[string]any{}, http.StatusOK)
	postJSON(t, other.app, other.ownerCookie, "/api/events/"+otherEventID+"/archive/notes", map[string]any{"body": "Move doors earlier."}, http.StatusOK)

	scopedSearch := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives?q=night", http.StatusOK)
	if matches, ok := scopedSearch.JSON.([]any); !ok || len(matches) != 1 || mustObject(t, matches[0])["eventId"] != firstEventID {
		t.Fatalf("expected workspace-scoped search to exclude other workspace archives, got %#v", scopedSearch.JSON)
	}

	tooLongQuery := strings.Repeat("a", 121)
	getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/archives?q="+tooLongQuery, http.StatusBadRequest)

	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusOK)
	getJSON(t, fx.app, nil, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusForbidden)
	getJSON(t, fx.app, other.memberCookie, "/api/workspaces/"+fx.workspaceID+"/archives", http.StatusForbidden)
}

func TestFirstEventLifecycleArchiveNotes(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 4)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "Move doors earlier."}, http.StatusOK)
	resp := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "Keep card reader charged."}, http.StatusOK)
	archive := mustObject(t, resp.JSON)
	if int(archive["noteCount"].(float64)) != 2 {
		t.Fatalf("expected two notes, got %#v", archive)
	}
	notes := archive["notes"].([]any)
	if mustObject(t, notes[0])["body"] != "Move doors earlier." || mustObject(t, notes[1])["body"] != "Keep card reader charged." {
		t.Fatalf("notes not ordered by creation: %#v", notes)
	}

	reloaded := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive", http.StatusOK)
	reloadedNotes := mustObject(t, reloaded.JSON)["notes"].([]any)
	if len(reloadedNotes) != 2 || mustObject(t, reloadedNotes[0])["body"] != "Move doors earlier." || mustObject(t, reloadedNotes[1])["body"] != "Keep card reader charged." {
		t.Fatalf("GET archive did not return persisted notes in order: %#v", reloadedNotes)
	}

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "   "}, http.StatusBadRequest)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "member note"}, http.StatusForbidden)
}

func TestFirstEventLifecycleArchiveSeedsDraftWithoutPrivateData(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 40, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/notes", map[string]any{"body": "Private lesson"}, http.StatusOK)

	startsAt, err := time.Parse(time.RFC3339Nano, mustString(t, event, "startsAt"))
	if err != nil {
		t.Fatal(err)
	}

	seeded := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/seed-draft", map[string]any{}, http.StatusOK)
	draft := mustObject(t, seeded.JSON)
	draftID := mustString(t, seeded.JSON, "id")
	expectedStartsAt := startsAt.AddDate(0, 0, 7).UTC().Format(time.RFC3339Nano)
	if draftID == eventID || draft["status"] != "draft" || draft["title"] != event["title"] || draft["workspaceId"] != fx.workspaceID || draft["startsAt"] != expectedStartsAt {
		t.Fatalf("unexpected seeded draft: %#v", draft)
	}
	if draft["publicSlug"] != nil || draft["publicUrl"] != nil {
		t.Fatalf("seeded draft must not have public URLs: %#v", draft)
	}
	archive := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive", http.StatusOK).JSON)
	if archive["seededEventId"] != draftID {
		t.Fatalf("expected archive detail to expose seeded draft id, got %#v", archive["seededEventId"])
	}

	seededAgain := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/archive/seed-draft", map[string]any{}, http.StatusOK)
	if mustString(t, seededAgain.JSON, "id") != draftID {
		t.Fatalf("expected seed draft retry to return same event id: first=%s second=%s", draftID, mustString(t, seededAgain.JSON, "id"))
	}

	var workspaceID, title, publicDescription, locationDisplay, pricingMode, ticketCurrency, status, createdByPersonID string
	var ticketAllocation, ticketPriceCents int
	var seededStartsAt time.Time
	var publicSlug sql.NullString
	var publishedAt, endedAt sql.NullTime
	if err := fx.app.db.QueryRow(t.Context(), `
		select workspace_id, title, starts_at, public_description, location_display,
		       ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
		       status, public_slug, published_at, ended_at, created_by_person_id
		from events
		where id = $1
	`, draftID).Scan(&workspaceID, &title, &seededStartsAt, &publicDescription, &locationDisplay, &ticketAllocation, &pricingMode, &ticketPriceCents, &ticketCurrency, &status, &publicSlug, &publishedAt, &endedAt, &createdByPersonID); err != nil {
		t.Fatal(err)
	}
	if workspaceID != fx.workspaceID || title != event["title"] || !seededStartsAt.Equal(startsAt.AddDate(0, 0, 7)) || publicDescription != event["publicDescription"].(string) || locationDisplay != event["locationDisplay"].(string) || ticketAllocation != 40 || pricingMode != "fixed" || ticketPriceCents != 1500 || ticketCurrency != "usd" || status != "draft" || publicSlug.Valid || publishedAt.Valid || endedAt.Valid || createdByPersonID != ownerPersonID(t, fx) {
		t.Fatalf("seeded event row mismatch: workspace=%q title=%q startsAt=%s description=%q location=%q allocation=%d pricing=%q price=%d currency=%q status=%q publicSlug=%v publishedAt=%v endedAt=%v createdBy=%q", workspaceID, title, seededStartsAt.Format(time.RFC3339Nano), publicDescription, locationDisplay, ticketAllocation, pricingMode, ticketPriceCents, ticketCurrency, status, publicSlug, publishedAt, endedAt, createdByPersonID)
	}

	var ticketCount, reportCount, settlementCount, archiveCount int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from tickets where event_id = $1`, draftID).Scan(&ticketCount); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_reports where event_id = $1`, draftID).Scan(&reportCount); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_settlements where event_id = $1`, draftID).Scan(&settlementCount); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_archives where event_id = $1`, draftID).Scan(&archiveCount); err != nil {
		t.Fatal(err)
	}
	if ticketCount != 0 || reportCount != 0 || settlementCount != 0 || archiveCount != 0 {
		t.Fatalf("seeded draft copied private rows: tickets=%d reports=%d settlements=%d archives=%d", ticketCount, reportCount, settlementCount, archiveCount)
	}

	var auditAction, auditSubjectType, auditSubjectID, auditEventID, auditNewEventID, auditWorkspaceID string
	if err := fx.app.db.QueryRow(t.Context(), `
		select action, subject_type, coalesce(subject_id::text, ''), metadata->>'eventId', metadata->>'newEventId', metadata->>'workspaceId'
		from audit_entries
		where action = 'archive.seed_draft_created'
		order by created_at desc
		limit 1
	`).Scan(&auditAction, &auditSubjectType, &auditSubjectID, &auditEventID, &auditNewEventID, &auditWorkspaceID); err != nil {
		t.Fatal(err)
	}
	if auditAction != "archive.seed_draft_created" || auditSubjectType != "event_archive" || auditEventID != eventID || auditNewEventID != draftID || auditWorkspaceID != fx.workspaceID {
		t.Fatalf("unexpected audit entry: action=%q subjectType=%q subjectID=%q eventId=%q newEventId=%q workspaceId=%q", auditAction, auditSubjectType, auditSubjectID, auditEventID, auditNewEventID, auditWorkspaceID)
	}

	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/archive/seed-draft", map[string]any{}, http.StatusForbidden)
	postJSON(t, fx.app, nil, "/api/events/"+eventID+"/archive/seed-draft", map[string]any{}, http.StatusForbidden)

	otherEvent := createEvent(t, fx, "Second Night", 10)
	otherEventID := mustString(t, otherEvent, "id")
	publishEvent(t, fx, otherEventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+otherEventID+"/archive/seed-draft", map[string]any{}, http.StatusNotFound)
}

func TestFirstEventLifecycleSettlementAdjustments(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 4, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("paid"), "Paid Guest", "paid", 1500, "usd")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	first := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{
		"amountCents": 500,
		"label":       "Donation",
		"reason":      "  community support  ",
	}, http.StatusOK)
	firstSettlement := mustObject(t, first.JSON)
	if int(firstSettlement["adjustmentTotalCents"].(float64)) != 500 || int(firstSettlement["netTotalCents"].(float64)) != 2000 {
		t.Fatalf("unexpected first adjustment totals: %#v", firstSettlement)
	}
	firstAdjustments := firstSettlement["adjustments"].([]any)
	if len(firstAdjustments) != 1 {
		t.Fatalf("expected one adjustment, got %#v", firstAdjustments)
	}
	firstAdjustment := mustObject(t, firstAdjustments[0])
	if firstAdjustment["amountCents"] != float64(500) || firstAdjustment["label"] != "Donation" || firstAdjustment["reason"] != "community support" || firstAdjustment["createdByPersonId"] != ownerPersonID(t, fx) {
		t.Fatalf("unexpected first adjustment row: %#v", firstAdjustment)
	}
	if firstAdjustment["settlementId"] == "" || firstAdjustment["createdAt"] == "" || firstAdjustment["id"] == "" {
		t.Fatalf("expected first adjustment identifiers: %#v", firstAdjustment)
	}

	time.Sleep(10 * time.Millisecond)
	second := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{
		"amountCents": -200,
		"label":       "Refund",
		"reason":      "oops",
	}, http.StatusOK)
	secondSettlement := mustObject(t, second.JSON)
	if int(secondSettlement["adjustmentTotalCents"].(float64)) != 300 || int(secondSettlement["netTotalCents"].(float64)) != 1800 {
		t.Fatalf("unexpected second adjustment totals: %#v", secondSettlement)
	}
	secondAdjustments := secondSettlement["adjustments"].([]any)
	if len(secondAdjustments) != 2 {
		t.Fatalf("expected two adjustments, got %#v", secondAdjustments)
	}
	if mustObject(t, secondAdjustments[0])["label"] != "Donation" || mustObject(t, secondAdjustments[1])["label"] != "Refund" {
		t.Fatalf("expected adjustments sorted by createdAt asc: %#v", secondAdjustments)
	}
	if int(mustObject(t, secondAdjustments[1])["amountCents"].(float64)) != -200 {
		t.Fatalf("unexpected second adjustment row: %#v", secondAdjustments[1])
	}

	memberSettlement := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/settlement", http.StatusOK)
	if !reflect.DeepEqual(second.JSON, memberSettlement.JSON) {
		t.Fatalf("expected member GET to match owner settlement after adjustments")
	}
}

func TestFirstEventLifecycleSettlementAdjustmentValidation(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 2)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("paid"), "Paid Guest", "paid", 1500, "usd")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{"amountCents": 0, "label": "Zero"}, http.StatusBadRequest)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{"amountCents": 100, "label": "   "}, http.StatusBadRequest)
}

func TestFirstEventLifecycleSettlementAdjustmentRequiresSettlement(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 2)
	eventID := mustString(t, event, "id")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/settlement/adjustments", map[string]any{"amountCents": 100, "label": "Donation"}, http.StatusNotFound)
}

func TestFirstEventLifecycleEndOfNightRejectsPendingPaidTickets(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("pending"), "Pending Guest", "pending", 1500, "usd")

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusConflict)
}

func TestFirstEventLifecyclePaidReportSettlementSummaryIsIdempotent(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 6, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("paid-a"), "Paid A", "paid", 1500, "usd")
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("paid-b"), "Paid B", "paid", 1500, "usd")
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("cancelled"), "Cancelled Guest", "cancelled", 1500, "usd")
	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("free"), "Free Guest", "free", 0, "usd")

	first := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	firstSummary := mustObject(t, mustObject(t, first.JSON)["settlementSummary"])
	if firstSummary["currency"] != "usd" || int(firstSummary["grossPaidRevenueCents"].(float64)) != 3000 || int(firstSummary["paidTicketCount"].(float64)) != 2 || int(firstSummary["pendingTicketCount"].(float64)) != 0 || int(firstSummary["cancelledTicketCount"].(float64)) != 1 || int(firstSummary["freeTicketCount"].(float64)) != 1 || int(firstSummary["reservedCount"].(float64)) != 3 {
		t.Fatalf("unexpected paid settlement summary: %#v", firstSummary)
	}

	type settlementRow struct {
		Currency              string
		GrossPaidRevenueCents int
		PaidTicketCount       int
		PendingTicketCount    int
		CancelledTicketCount  int
		FreeTicketCount       int
		ReservedCount         int
		Status                string
		GeneratedByPersonID   string
		GeneratedAt           time.Time
		CreatedAt             time.Time
		UpdatedAt             time.Time
	}
	settlement := settlementRow{}
	if err := fx.app.db.QueryRow(t.Context(), `
		select currency, gross_paid_revenue_cents, paid_ticket_count, pending_ticket_count,
		       cancelled_ticket_count, free_ticket_count, reserved_count, status,
		       generated_by_person_id, generated_at, created_at, updated_at
		from event_settlements
		where event_id = $1
	`, eventID).Scan(&settlement.Currency, &settlement.GrossPaidRevenueCents, &settlement.PaidTicketCount, &settlement.PendingTicketCount, &settlement.CancelledTicketCount, &settlement.FreeTicketCount, &settlement.ReservedCount, &settlement.Status, &settlement.GeneratedByPersonID, &settlement.GeneratedAt, &settlement.CreatedAt, &settlement.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	if settlement.Currency != "usd" || settlement.GrossPaidRevenueCents != 3000 || settlement.PaidTicketCount != 2 || settlement.PendingTicketCount != 0 || settlement.CancelledTicketCount != 1 || settlement.FreeTicketCount != 1 || settlement.ReservedCount != 3 || settlement.Status != "open" || settlement.GeneratedByPersonID != ownerPersonID(t, fx) {
		t.Fatalf("unexpected settlement row: %#v", settlement)
	}

	insertTicketWithPaymentStatus(t, fx, eventID, fx.email("late-paid"), "Late Paid", "paid", 9999, "usd")
	second := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	if !reflect.DeepEqual(first.JSON, second.JSON) {
		t.Fatalf("expected end-of-night report to be idempotent: first=%#v second=%#v", first.JSON, second.JSON)
	}

	settlementAfter := settlementRow{}
	if err := fx.app.db.QueryRow(t.Context(), `
		select currency, gross_paid_revenue_cents, paid_ticket_count, pending_ticket_count,
		       cancelled_ticket_count, free_ticket_count, reserved_count, status,
		       generated_by_person_id, generated_at, created_at, updated_at
		from event_settlements
		where event_id = $1
	`, eventID).Scan(&settlementAfter.Currency, &settlementAfter.GrossPaidRevenueCents, &settlementAfter.PaidTicketCount, &settlementAfter.PendingTicketCount, &settlementAfter.CancelledTicketCount, &settlementAfter.FreeTicketCount, &settlementAfter.ReservedCount, &settlementAfter.Status, &settlementAfter.GeneratedByPersonID, &settlementAfter.GeneratedAt, &settlementAfter.CreatedAt, &settlementAfter.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(settlement, settlementAfter) {
		t.Fatalf("expected settlement row to remain unchanged: first=%#v second=%#v", settlement, settlementAfter)
	}
}

func TestFirstEventLifecycleCreatesArchiveAtEndOfNight(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 4)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)

	var archiveID string
	var status string
	var noteCount int
	if err := fx.app.db.QueryRow(t.Context(), `
		select id, status, note_count
		from event_archives
		where event_id = $1
	`, eventID).Scan(&archiveID, &status, &noteCount); err != nil {
		t.Fatal(err)
	}
	if archiveID == "" || status != "private" || noteCount != 0 {
		t.Fatalf("unexpected archive row: id=%q status=%q noteCount=%d", archiveID, status, noteCount)
	}

	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_archives where event_id = $1`, eventID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one archive row, got %d", count)
	}
}

func TestFirstEventLifecycleOldReportSnapshotOmitsSettlementSummary(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 2)
	eventID := mustString(t, event, "id")
	published := publishEvent(t, fx, eventID)
	ownerID := ownerPersonID(t, fx)
	oldSnapshot := map[string]any{
		"id":                     "legacy-report",
		"eventId":                eventID,
		"title":                  "Night Market",
		"startsAt":               event["startsAt"],
		"publicUrl":              published["publicUrl"],
		"ticketAllocation":       2,
		"ticketsReserved":        0,
		"ticketsCheckedIn":       0,
		"noShows":                0,
		"generatedAt":            time.Now().UTC().Format(time.RFC3339Nano),
		"generatedByMemberEmail": fx.email("owner"),
	}
	payload, err := json.Marshal(oldSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.app.db.Exec(t.Context(), `
		insert into event_reports (event_id, generated_by_person_id, snapshot)
		values ($1, $2, $3)
	`, eventID, ownerID, payload); err != nil {
		t.Fatal(err)
	}

	first := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	reportObj := mustObject(t, first.JSON)
	if _, ok := reportObj["settlementSummary"]; ok {
		t.Fatalf("legacy snapshot should not synthesize settlement summary: %#v", reportObj)
	}

	second := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	if !reflect.DeepEqual(first.JSON, second.JSON) {
		t.Fatalf("expected legacy end-of-night retry to return same snapshot: first=%#v second=%#v", first.JSON, second.JSON)
	}

	var archiveCount, settlementCount int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_archives where event_id = $1`, eventID).Scan(&archiveCount); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_settlements where event_id = $1`, eventID).Scan(&settlementCount); err != nil {
		t.Fatal(err)
	}
	if archiveCount != 0 || settlementCount != 0 {
		t.Fatalf("legacy retry should not require archive or settlement: archives=%d settlements=%d", archiveCount, settlementCount)
	}
}

func TestReadyWithDatabase(t *testing.T) {
	fx := newLifecycleFixture(t)
	getJSON(t, fx.app, nil, "/api/ready", http.StatusOK)
}

func TestFirstEventLifecycleFixedPriceCreate(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	if event["pricingMode"] != "fixed" || int(event["ticketPriceCents"].(float64)) != 1500 || event["ticketCurrency"] != "usd" {
		t.Fatalf("unexpected pricing response: %#v", event)
	}
}

func TestFirstEventLifecycleRejectsLowFixedPrice(t *testing.T) {
	fx := newLifecycleFixture(t)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{
		"title":             "Night Market",
		"startsAt":          "2026-07-01T20:00:00Z",
		"publicDescription": "Free community event.",
		"locationDisplay":   "Warehouse District",
		"ticketAllocation":  2,
		"pricingMode":       "fixed",
		"ticketPriceCents":  49,
		"ticketCurrency":    "usd",
	}, http.StatusBadRequest)
}

func TestFirstEventLifecycleRejectsPricingChangeAfterReservation(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusOK)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID, map[string]any{
		"pricingMode":      "fixed",
		"ticketPriceCents": 1500,
		"ticketCurrency":   "usd",
	}, http.StatusConflict)
}

func TestFirstEventLifecycleFixedPriceRequiresPaidCheckout(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	slug := mustString(t, publishEvent(t, fx, mustString(t, event, "id")), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusConflict)
}

func TestPaidReservationWithoutPaymentProviderReturns503(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	slug := mustString(t, publishEvent(t, fx, mustString(t, event, "id")), "publicSlug")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusServiceUnavailable)
}

func TestPaidReservationCreatesPendingTicketAndCheckoutSession(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	fake := &fakePaymentProvider{response: checkoutSessionResponse{ID: "cs_test_" + strings.ReplaceAll(eventID, "-", ""), URL: "https://checkout.example/session"}}
	fx.app.payments = fake
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	email := fx.email("guest")
	resp := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/paid-reservations", map[string]any{"email": email, "displayName": "Guest"}, http.StatusOK)
	if mustString(t, resp.JSON, "checkoutSessionId") != fake.response.ID || mustString(t, resp.JSON, "checkoutUrl") != "https://checkout.example/session" {
		t.Fatalf("unexpected checkout response: %#v", resp.JSON)
	}

	if fake.request.TicketID == "" || fake.request.EventID != eventID || fake.request.EventTitle != "Night Market" || fake.request.AmountCents != 1500 || fake.request.Currency != "usd" {
		t.Fatalf("unexpected provider request: %#v", fake.request)
	}
	if !strings.Contains(fake.request.SuccessURL, "/tickets/") || !strings.Contains(fake.request.SuccessURL, "checkout=success") {
		t.Fatalf("unexpected success url: %s", fake.request.SuccessURL)
	}
	if !strings.Contains(fake.request.CancelURL, "/e/") || !strings.Contains(fake.request.CancelURL, "checkout=cancelled") {
		t.Fatalf("unexpected cancel url: %s", fake.request.CancelURL)
	}

	var paymentStatus, currency, sessionID, status string
	var amountCents int
	if err := fx.app.db.QueryRow(t.Context(), `
		select payment_status, amount_cents, currency, stripe_checkout_session_id, status
		from tickets
		where event_id = $1 and email = $2
	`, eventID, email).Scan(&paymentStatus, &amountCents, &currency, &sessionID, &status); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "pending" || amountCents != 1500 || currency != "usd" || sessionID != fake.response.ID || status != "reserved" {
		t.Fatalf("unexpected ticket state: paymentStatus=%s amount=%d currency=%s session=%s status=%s", paymentStatus, amountCents, currency, sessionID, status)
	}
}

func TestFreeReservationStillWorksForFreeEvent(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 2)
	slug := mustString(t, publishEvent(t, fx, mustString(t, event, "id")), "publicSlug")
	resp := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusOK)
	if mustString(t, resp.JSON, "status") != "reserved" {
		t.Fatalf("unexpected reservation response: %#v", resp.JSON)
	}
}

func TestStripeWebhookFulfillmentMarksTicketPaidAndEnqueuesEmail(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	ticketID, sessionID := insertPendingStripeTicket(t, fx, eventID, "guest@example.test", "Guest", testStripeSessionID(t, "cs_test_fulfillment"))
	stripeEventID := testStripeEventID(t, "evt_fulfillment")

	eventPayload := stripeCheckoutSessionEvent(t, stripe.EventTypeCheckoutSessionCompleted, stripeEventID, sessionID, ticketID, eventID, stripe.CheckoutSessionPaymentStatusPaid, 1500, "usd")
	if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
		t.Fatal(err)
	}

	assertPaidTicketState(t, fx, ticketID, sessionID)
	assertEmailOutboxCount(t, fx, ticketID, 1)
	assertWebhookEventCount(t, fx, stripeEventID, 1)
}

func TestStripeWebhookFulfillmentIsIdempotentForDuplicateEventID(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	ticketID, sessionID := insertPendingStripeTicket(t, fx, eventID, "guest@example.test", "Guest", testStripeSessionID(t, "cs_test_idempotent"))
	stripeEventID := testStripeEventID(t, "evt_duplicate")

	eventPayload := stripeCheckoutSessionEvent(t, stripe.EventTypeCheckoutSessionCompleted, stripeEventID, sessionID, ticketID, eventID, stripe.CheckoutSessionPaymentStatusPaid, 1500, "usd")
	if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
		t.Fatal(err)
	}

	assertPaidTicketState(t, fx, ticketID, sessionID)
	assertEmailOutboxCount(t, fx, ticketID, 1)
	assertWebhookEventCount(t, fx, stripeEventID, 1)
}

func TestStripeWebhookCompletedRejectsMismatchedPaymentDetails(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	cases := []struct {
		name          string
		paymentStatus stripe.CheckoutSessionPaymentStatus
		amountTotal   int64
		currency      string
	}{
		{name: "unpaid-status", paymentStatus: stripe.CheckoutSessionPaymentStatusUnpaid, amountTotal: 1500, currency: "usd"},
		{name: "wrong-amount", paymentStatus: stripe.CheckoutSessionPaymentStatusPaid, amountTotal: 1600, currency: "usd"},
		{name: "wrong-currency", paymentStatus: stripe.CheckoutSessionPaymentStatusPaid, amountTotal: 1500, currency: "eur"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stripeEventID := testStripeEventID(t, "evt_"+tc.name)
			ticketID, sessionID := insertPendingStripeTicket(t, fx, eventID, "guest+"+tc.name+"@example.test", "Guest "+tc.name, testStripeSessionID(t, "cs_test_"+tc.name))
			eventPayload := stripeCheckoutSessionEvent(t, stripe.EventTypeCheckoutSessionCompleted, stripeEventID, sessionID, ticketID, eventID, tc.paymentStatus, tc.amountTotal, tc.currency)
			if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
				t.Fatal(err)
			}

			var paymentStatus string
			var paidAt sql.NullTime
			if err := fx.app.db.QueryRow(t.Context(), `select payment_status, paid_at from tickets where id = $1`, ticketID).Scan(&paymentStatus, &paidAt); err != nil {
				t.Fatal(err)
			}
			if paymentStatus != "pending" || paidAt.Valid {
				t.Fatalf("ticket should remain pending for %s: status=%s paidAt=%v", tc.name, paymentStatus, paidAt)
			}
			assertEmailOutboxCount(t, fx, ticketID, 0)
			assertWebhookEventCount(t, fx, stripeEventID, 1)
		})
	}
}

func TestStripeWebhookWrongSessionIDDoesNotMarkTicketPaid(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	ticketID, _ := insertPendingStripeTicket(t, fx, eventID, "guest@example.test", "Guest", testStripeSessionID(t, "cs_test_expected"))
	stripeEventID := testStripeEventID(t, "evt_wrong_session")

	eventPayload := stripeCheckoutSessionEvent(t, stripe.EventTypeCheckoutSessionCompleted, stripeEventID, testStripeSessionID(t, "cs_test_wrong"), ticketID, eventID, stripe.CheckoutSessionPaymentStatusPaid, 1500, "usd")
	if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
		t.Fatal(err)
	}

	var paymentStatus string
	var paidAt sql.NullTime
	if err := fx.app.db.QueryRow(t.Context(), `select payment_status, paid_at from tickets where id = $1`, ticketID).Scan(&paymentStatus, &paidAt); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "pending" || paidAt.Valid {
		t.Fatalf("ticket should remain pending on session mismatch: status=%s paidAt=%v", paymentStatus, paidAt)
	}
	assertEmailOutboxCount(t, fx, ticketID, 0)
	assertWebhookEventCount(t, fx, stripeEventID, 1)
}

func TestStripeWebhookExpiredCancelsPendingTicketAndReleasesCapacity(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 1, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	published := publishEvent(t, fx, eventID)
	slug := mustString(t, published, "publicSlug")
	ticketID, sessionID := insertPendingStripeTicket(t, fx, eventID, "guest@example.test", "Guest", testStripeSessionID(t, "cs_test_expired"))
	stripeEventID := testStripeEventID(t, "evt_expired")

	publicBefore := getJSON(t, fx.app, nil, "/api/public/events/"+slug, http.StatusOK).JSON.(map[string]any)
	if int(publicBefore["remainingTickets"].(float64)) != 0 {
		t.Fatalf("expected pending ticket to consume capacity before expiration: %#v", publicBefore)
	}

	eventPayload := stripeCheckoutSessionEvent(t, stripe.EventTypeCheckoutSessionExpired, stripeEventID, sessionID, ticketID, eventID, stripe.CheckoutSessionPaymentStatusUnpaid, 1500, "usd")
	if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
		t.Fatal(err)
	}

	var paymentStatus string
	if err := fx.app.db.QueryRow(t.Context(), `select payment_status from tickets where id = $1`, ticketID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "cancelled" {
		t.Fatalf("expected expired ticket to be cancelled, got %s", paymentStatus)
	}
	publicAfter := getJSON(t, fx.app, nil, "/api/public/events/"+slug, http.StatusOK).JSON.(map[string]any)
	if int(publicAfter["remainingTickets"].(float64)) != 1 || publicAfter["isFull"].(bool) != false {
		t.Fatalf("expected capacity to be released after expiration: %#v", publicAfter)
	}
	assertWebhookEventCount(t, fx, stripeEventID, 1)
}

func TestStripeWebhookCompletedDoesNotFulfillAfterEndOfNight(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
	ticketID, sessionID := insertPendingStripeTicket(t, fx, eventID, "guest@example.test", "Guest", testStripeSessionID(t, "cs_test_after_close"))
	stripeEventID := testStripeEventID(t, "evt_after_close")

	eventPayload := stripeCheckoutSessionEvent(t, stripe.EventTypeCheckoutSessionCompleted, stripeEventID, sessionID, ticketID, eventID, stripe.CheckoutSessionPaymentStatusPaid, 1500, "usd")
	if err := fx.app.processStripeWebhookEvent(t.Context(), eventPayload); err != nil {
		t.Fatal(err)
	}

	var paymentStatus string
	var paidAt sql.NullTime
	if err := fx.app.db.QueryRow(t.Context(), `select payment_status, paid_at from tickets where id = $1`, ticketID).Scan(&paymentStatus, &paidAt); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "pending" || paidAt.Valid {
		t.Fatalf("ticket should remain pending after close: status=%s paidAt=%v", paymentStatus, paidAt)
	}
	assertEmailOutboxCount(t, fx, ticketID, 0)
	assertWebhookEventCount(t, fx, stripeEventID, 1)
}

func TestDoorTicketSearchExcludesPendingPaidTicket(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	published := publishEvent(t, fx, eventID)
	slug := mustString(t, published, "publicSlug")
	guestEmail := fx.email("guest-search")
	insertPendingStripeTicket(t, fx, eventID, guestEmail, "Guest Search", testStripeSessionID(t, "cs_test_search"))

	search := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/tickets?query=guest", http.StatusOK)
	if len(search.JSON.([]any)) != 0 {
		t.Fatalf("expected pending paid ticket to be excluded from search: %#v", search.JSON)
	}

	publicEvent := getJSON(t, fx.app, nil, "/api/public/events/"+slug, http.StatusOK).JSON.(map[string]any)
	if int(publicEvent["remainingTickets"].(float64)) != 1 {
		t.Fatalf("expected pending ticket to count toward capacity before expiry: %#v", publicEvent)
	}
}

func TestDoorCheckInRejectsPendingPaidTicket(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEventWithPricing(t, fx, "Night Market", 2, "fixed", 1500, "usd")
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)
	ticketID, _ := insertPendingStripeTicket(t, fx, eventID, fx.email("guest-checkin"), "Guest Checkin", testStripeSessionID(t, "cs_test_checkin"))
	var code string
	if err := fx.app.db.QueryRow(t.Context(), `select code from tickets where id = $1`, ticketID).Scan(&code); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": code}, http.StatusConflict)

	var paymentStatus string
	if err := fx.app.db.QueryRow(t.Context(), `select payment_status from tickets where id = $1`, ticketID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "pending" {
		t.Fatalf("expected pending ticket to remain pending after rejected check-in, got %s", paymentStatus)
	}
}

func TestFirstEventLifecycleFullCapacity(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")

	firstGuest := fx.email("guest-a")
	secondGuest := fx.email("guest-b")
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": firstGuest, "displayName": "Guest One"}, http.StatusOK)
	postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": secondGuest, "displayName": "Guest Two"}, http.StatusConflict)

	publicEvent := getJSON(t, fx.app, nil, "/api/public/events/"+slug, http.StatusOK)
	if int(publicEvent.JSON.(map[string]any)["reservedCount"].(float64)) != 1 || int(publicEvent.JSON.(map[string]any)["remainingTickets"].(float64)) != 0 || publicEvent.JSON.(map[string]any)["isFull"].(bool) != true {
		t.Fatalf("event should remain full with one reservation: %#v", publicEvent.JSON)
	}
}

func TestFirstEventLifecycleDuplicateCheckIn(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")
	ticket := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusOK)
	code := mustString(t, ticket.JSON, "code")
	firstCheckIn := postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": code}, http.StatusOK)
	secondCheckIn := postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": code}, http.StatusOK)

	first := firstCheckIn.JSON.(map[string]any)
	second := secondCheckIn.JSON.(map[string]any)
	if first["status"] != "checked_in" || second["status"] != "checked_in" {
		t.Fatalf("expected checked-in ticket: first=%#v second=%#v", firstCheckIn.JSON, secondCheckIn.JSON)
	}
	if first["checkedInAt"] != second["checkedInAt"] {
		t.Fatalf("expected duplicate check-in to return same checkedInAt, got %v and %v", first["checkedInAt"], second["checkedInAt"])
	}
}

func TestFirstEventLifecycleDraftReservationsBlocked(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")
	if event["publicSlug"] != nil {
		t.Fatalf("draft event should not have a public slug: %#v", event["publicSlug"])
	}

	getJSON(t, fx.app, nil, "/api/public/events/"+eventID, http.StatusNotFound)
	postJSON(t, fx.app, nil, "/api/public/events/"+eventID+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusNotFound)
}

func TestFirstEventLifecyclePermissions(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Night Market", 1)
	eventID := mustString(t, event, "id")

	updated := patchJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID, map[string]any{"title": "Night Market Updated"}, http.StatusOK)
	if mustString(t, updated.JSON, "title") != "Night Market Updated" {
		t.Fatalf("member update did not persist: %#v", updated.JSON)
	}

	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/publish", map[string]any{}, http.StatusForbidden)
	published := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/publish", map[string]any{}, http.StatusOK)
	slug := mustString(t, published.JSON, "publicSlug")
	guestTicket := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/reservations", map[string]any{"email": fx.email("guest"), "displayName": "Guest"}, http.StatusOK)
	search := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/tickets?query=guest", http.StatusOK)
	tickets := search.JSON.([]any)
	if len(tickets) != 1 {
		t.Fatalf("expected one ticket in door search, got %#v", search.JSON)
	}
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/door/check-ins", map[string]any{"code": mustString(t, guestTicket.JSON, "code")}, http.StatusOK)
	postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusForbidden)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/end-of-night", map[string]any{}, http.StatusOK)
}

func TestDevEmailOutbox(t *testing.T) {
	fx := newLifecycleFixture(t)
	inviteEmail := fx.email("invitee")
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": inviteEmail}, http.StatusOK)

	outbox := getJSON(t, fx.app, fx.ownerCookie, "/api/dev/email-outbox", http.StatusOK).JSON.([]any)
	if len(outbox) == 0 {
		t.Fatal("expected at least one email in outbox")
	}

	matched := false
	for _, item := range outbox {
		msg := mustObject(t, item)
		if msg["recipientEmail"] == inviteEmail && strings.Contains(msg["body"].(string), "/invite/") {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("expected invitation email for %s in outbox: %#v", inviteEmail, outbox)
	}
}

func TestDevEmailOutboxUnauthorized(t *testing.T) {
	fx := newLifecycleFixture(t)
	getJSON(t, fx.app, nil, "/api/dev/email-outbox", http.StatusUnauthorized)
}

func TestDevEmailOutboxProductionHidden(t *testing.T) {
	fx := newLifecycleFixture(t)
	prodApp := New(Config{AppEnv: "production", PublicWebURL: "http://public.test", SessionSecret: "test-secret"}, fx.app.db)
	getJSON(t, prodApp, fx.ownerCookie, "/api/dev/email-outbox", http.StatusNotFound)
}

func TestGetSpecificWorkspace(t *testing.T) {
	fx := newLifecycleFixture(t)
	second := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces", map[string]any{"name": "Second Room"}, http.StatusOK)
	secondID := mustString(t, second.JSON, "id")

	loaded := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+secondID, http.StatusOK).JSON
	if mustString(t, loaded, "id") != secondID || mustString(t, loaded, "name") != "Second Room" {
		t.Fatalf("unexpected workspace response: %#v", loaded)
	}

	otherCookie := postJSON(t, fx.app, nil, "/api/auth/signup", map[string]any{"email": fx.email("other"), "password": "secret1234", "displayName": "Other"}, http.StatusOK).Cookie
	getJSON(t, fx.app, otherCookie, "/api/workspaces/"+secondID, http.StatusForbidden)
}

func TestSignupStoresBcryptPasswordHash(t *testing.T) {
	fx := newLifecycleFixture(t)
	email := fx.email("bcrypt")
	postJSON(t, fx.app, nil, "/api/auth/signup", map[string]any{"email": email, "password": "secret1234", "displayName": "Hash"}, http.StatusOK)

	var storedHash string
	if err := fx.app.db.QueryRow(t.Context(), `select password_hash from people where email = $1`, email).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storedHash, passwordScheme+"$") || strings.Contains(storedHash, legacySHA256Scheme+"$") {
		t.Fatalf("expected bcrypt password hash, got %q", storedHash)
	}
	valid, upgraded := verifyPassword("secret1234", storedHash)
	if !valid || upgraded != "" {
		t.Fatalf("expected bcrypt password to verify without upgrade, valid=%v upgraded=%q", valid, upgraded)
	}
}

func TestLegacyPasswordHashUpgradesOnLogin(t *testing.T) {
	fx := newLifecycleFixture(t)
	email := fx.email("legacy")
	legacyHash := legacyPasswordHashForTest(t, "secret1234")
	if _, err := fx.app.db.Exec(t.Context(), `insert into people (email, password_hash) values ($1, $2)`, email, legacyHash); err != nil {
		t.Fatal(err)
	}

	postJSON(t, fx.app, nil, "/api/auth/login", map[string]any{"email": email, "password": "secret1234"}, http.StatusOK)

	var storedHash string
	if err := fx.app.db.QueryRow(t.Context(), `select password_hash from people where email = $1`, email).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(storedHash, passwordScheme+"$") {
		t.Fatalf("expected legacy hash to upgrade to bcrypt, got %q", storedHash)
	}
}

func TestOriginGuardRejectsCrossSiteCookieMutations(t *testing.T) {
	fx := newLifecycleFixture(t)
	postJSONWithOrigin(t, fx.app, fx.ownerCookie, "/api/workspaces", map[string]any{"name": "Evil"}, "http://evil.example", http.StatusForbidden)
	postJSONWithOrigin(t, fx.app, fx.ownerCookie, "/api/workspaces", map[string]any{"name": "Allowed"}, "http://public.test", http.StatusOK)
}

func TestLoginRateLimit(t *testing.T) {
	fx := newLifecycleFixture(t)
	email := fx.email("missing")
	for range maxLoginFailures {
		postJSON(t, fx.app, nil, "/api/auth/login", map[string]any{"email": email, "password": "wrong-password"}, http.StatusUnauthorized)
	}
	resp := postJSON(t, fx.app, nil, "/api/auth/login", map[string]any{"email": email, "password": "wrong-password"}, http.StatusTooManyRequests)
	if !strings.Contains(resp.Body, "too many login attempts") {
		t.Fatalf("expected rate-limit response, got %s", resp.Body)
	}
}

type testResponse struct {
	Status int
	Cookie *http.Cookie
	JSON   any
	Body   string
}

type lifecycleFixture struct {
	app          *App
	ownerCookie  *http.Cookie
	memberCookie *http.Cookie
	workspaceID  string
	suffix       string
}

func (f lifecycleFixture) email(prefix string) string {
	return prefix + "+" + f.suffix + "@example.test"
}

func newLifecycleFixture(t *testing.T) lifecycleFixture {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("set TEST_DATABASE_URL to run lifecycle acceptance test")
	}

	ctx := t.Context()
	db, err := OpenDB(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	app := New(Config{AppEnv: "test", PublicWebURL: "http://public.test", SessionSecret: "test-secret"}, db)
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-")) + fmt.Sprintf("-%d", time.Now().UnixNano())
	ownerEmail := "owner+" + suffix + "@example.test"
	memberEmail := "member+" + suffix + "@example.test"

	ownerCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email": ownerEmail, "password": "secret1234", "displayName": "Owner"}, http.StatusOK).Cookie
	workspace := postJSON(t, app, ownerCookie, "/api/workspaces", map[string]any{"name": "Signal Collective"}, http.StatusOK)
	workspaceID := mustString(t, workspace.JSON, "id")
	invite := postJSON(t, app, ownerCookie, "/api/workspaces/"+workspaceID+"/invitations", map[string]any{"email": memberEmail}, http.StatusOK)
	memberCookie := postJSON(t, app, nil, "/api/auth/signup", map[string]any{"email": memberEmail, "password": "secret1234", "displayName": "Door"}, http.StatusOK).Cookie
	postJSON(t, app, memberCookie, "/api/invitations/"+mustString(t, invite.JSON, "token")+"/accept", map[string]any{}, http.StatusOK)

	return lifecycleFixture{app: app, ownerCookie: ownerCookie, memberCookie: memberCookie, workspaceID: workspaceID, suffix: suffix}
}

type fakePaymentProvider struct {
	request  checkoutSessionRequest
	response checkoutSessionResponse
}

func (f *fakePaymentProvider) CreateCheckoutSession(_ context.Context, req checkoutSessionRequest) (checkoutSessionResponse, error) {
	f.request = req
	if f.response.ID == "" {
		f.response.ID = "cs_test_fake"
	}
	if f.response.URL == "" {
		f.response.URL = "https://checkout.example/fake"
	}
	return f.response, nil
}

func createEvent(t *testing.T, fx lifecycleFixture, title string, ticketAllocation int) map[string]any {
	t.Helper()
	return createEventWithPricing(t, fx, title, ticketAllocation, "free", 0, "usd")
}

func createEventWithPricing(t *testing.T, fx lifecycleFixture, title string, ticketAllocation int, pricingMode string, ticketPriceCents int, ticketCurrency string) map[string]any {
	t.Helper()
	resp := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": title, "startsAt": "2026-07-01T20:00:00Z", "publicDescription": "Free community event.", "locationDisplay": "Warehouse District", "ticketAllocation": ticketAllocation, "pricingMode": pricingMode, "ticketPriceCents": ticketPriceCents, "ticketCurrency": ticketCurrency}, http.StatusOK)
	return mustObject(t, resp.JSON)
}

func insertPendingStripeTicket(t *testing.T, fx lifecycleFixture, eventID, email, displayName, sessionID string) (string, string) {
	t.Helper()
	code, err := newTicketCode()
	if err != nil {
		t.Fatal(err)
	}
	var ticketID string
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into tickets (event_id, email, display_name, code, status, payment_status, amount_cents, currency, stripe_checkout_session_id)
		values ($1, $2, $3, $4, 'reserved', 'pending', 1500, 'usd', $5)
		returning id
	`, eventID, email, displayName, code, sessionID).Scan(&ticketID); err != nil {
		t.Fatal(err)
	}
	return ticketID, sessionID
}

func insertTicketWithPaymentStatus(t *testing.T, fx lifecycleFixture, eventID, email, displayName, paymentStatus string, amountCents int, currency string) string {
	t.Helper()
	code, err := newTicketCode()
	if err != nil {
		t.Fatal(err)
	}
	var ticketID string
	if err := fx.app.db.QueryRow(t.Context(), `
		insert into tickets (event_id, email, display_name, code, status, payment_status, amount_cents, currency)
		values ($1, $2, $3, $4, 'reserved', $5, $6, $7)
		returning id
	`, eventID, email, displayName, code, paymentStatus, amountCents, currency).Scan(&ticketID); err != nil {
		t.Fatal(err)
	}
	return ticketID
}

func ownerPersonID(t *testing.T, fx lifecycleFixture) string {
	t.Helper()
	var personID string
	if err := fx.app.db.QueryRow(t.Context(), `
		select person_id
		from workspace_members
		where workspace_id = $1 and role = 'owner'
		limit 1
	`, fx.workspaceID).Scan(&personID); err != nil {
		t.Fatal(err)
	}
	return personID
}

func stripeCheckoutSessionEvent(t *testing.T, eventType stripe.EventType, eventID, sessionID, ticketID, eventRefID string, paymentStatus stripe.CheckoutSessionPaymentStatus, amountTotal int64, currency string) stripe.Event {
	t.Helper()
	sessionPayload, err := json.Marshal(map[string]any{
		"id":             sessionID,
		"object":         "checkout.session",
		"payment_status": string(paymentStatus),
		"amount_total":   amountTotal,
		"currency":       strings.ToLower(currency),
		"metadata": map[string]string{
			"ticket_id": ticketID,
			"event_id":  eventRefID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return stripe.Event{
		ID:   eventID,
		Type: eventType,
		Data: &stripe.EventData{Raw: sessionPayload},
	}
}

func testStripeEventID(t *testing.T, prefix string) string {
	t.Helper()
	return testStripeID(t, prefix)
}

func testStripeSessionID(t *testing.T, prefix string) string {
	t.Helper()
	return testStripeID(t, prefix)
}

func testStripeID(t *testing.T, prefix string) string {
	t.Helper()
	suffix := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	return prefix + "_" + suffix + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
}

func assertPaidTicketState(t *testing.T, fx lifecycleFixture, ticketID, sessionID string) {
	t.Helper()
	var paymentStatus, storedSessionID string
	var paidAt sql.NullTime
	if err := fx.app.db.QueryRow(t.Context(), `
		select payment_status, coalesce(stripe_checkout_session_id, ''), paid_at
		from tickets
		where id = $1
	`, ticketID).Scan(&paymentStatus, &storedSessionID, &paidAt); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != "paid" || storedSessionID != sessionID || !paidAt.Valid {
		t.Fatalf("unexpected paid ticket state: status=%s session=%s paidAt=%v", paymentStatus, storedSessionID, paidAt)
	}
}

func assertEmailOutboxCount(t *testing.T, fx lifecycleFixture, ticketID string, want int) {
	t.Helper()
	var got int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from email_outbox where related_id = $1`, ticketID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("unexpected email count for %s: got %d want %d", ticketID, got, want)
	}
}

func assertWebhookEventCount(t *testing.T, fx lifecycleFixture, eventID string, want int) {
	t.Helper()
	var got int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from payment_webhook_events where id = $1`, eventID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("unexpected webhook event count for %s: got %d want %d", eventID, got, want)
	}
}

func publishEvent(t *testing.T, fx lifecycleFixture, eventID string) map[string]any {
	t.Helper()
	return mustObject(t, publishEventResult(t, fx.app, fx.ownerCookie, eventID).JSON)
}

func publishEventResult(t *testing.T, app *App, cookie *http.Cookie, eventID string) testResponse {
	t.Helper()
	return postJSON(t, app, cookie, "/api/events/"+eventID+"/publish", map[string]any{}, http.StatusOK)
}

func doJSON(t *testing.T, method string, app *App, cookie *http.Cookie, path string, payload any, wantStatus int) testResponse {
	t.Helper()
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s got %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var decoded any
	if strings.TrimSpace(rec.Body.String()) != "" {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
	}
	var cookieOut *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == authCookieName {
			cookieOut = c
		}
	}
	return testResponse{Status: rec.Code, Cookie: cookieOut, JSON: decoded, Body: rec.Body.String()}
}

func postJSON(t *testing.T, app *App, cookie *http.Cookie, path string, payload any, wantStatus int) testResponse {
	t.Helper()
	return doJSON(t, http.MethodPost, app, cookie, path, payload, wantStatus)
}

func postJSONWithOrigin(t *testing.T, app *App, cookie *http.Cookie, path string, payload any, origin string, wantStatus int) testResponse {
	t.Helper()
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Host = "public.test"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("POST %s origin %s got %d, want %d: %s", path, origin, rec.Code, wantStatus, rec.Body.String())
	}
	var decoded any
	if strings.TrimSpace(rec.Body.String()) != "" {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
	}
	return testResponse{Status: rec.Code, JSON: decoded, Body: rec.Body.String()}
}

func getJSON(t *testing.T, app *App, cookie *http.Cookie, path string, wantStatus int) testResponse {
	t.Helper()
	return doJSON(t, http.MethodGet, app, cookie, path, nil, wantStatus)
}

func patchJSON(t *testing.T, app *App, cookie *http.Cookie, path string, payload any, wantStatus int) testResponse {
	t.Helper()
	return doJSON(t, http.MethodPatch, app, cookie, path, payload, wantStatus)
}

func mustObject(t *testing.T, value any) map[string]any {
	t.Helper()
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object, got %#v", value)
	}
	return obj
}

func mustString(t *testing.T, value any, key string) string {
	t.Helper()
	obj := mustObject(t, value)
	v, ok := obj[key].(string)
	if !ok {
		t.Fatalf("expected %s string, got %#v", key, obj[key])
	}
	return v
}

func legacyPasswordHashForTest(t *testing.T, password string) string {
	t.Helper()
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s$%s$%s", legacySHA256Scheme, base64.RawURLEncoding.EncodeToString(salt), passwordSum(salt, password))
}
