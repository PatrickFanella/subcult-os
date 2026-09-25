package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOccurrenceLifecycleConcurrentPreviewHasOneWinner(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Concurrent edits", 10)
	path := "/api/events/" + mustString(t, event, "id") + "/occurrences"
	created := postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
		"name": "Concurrent occurrence", "startsAt": "2026-10-01T20:00:00Z",
	}, http.StatusOK)
	path += "/" + mustString(t, created.JSON, "id")
	handler := fx.app.Handler()
	ready := make(chan struct{})
	results := make(chan int, 2)
	for _, status := range []string{"cancelled", "postponed"} {
		payload, err := json.Marshal(map[string]any{"status": status, "expectedUpdatedAt": mustString(t, created.JSON, "updatedAt")})
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			<-ready
			req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(fx.ownerCookie)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			results <- rec.Code
		}()
	}
	close(ready)
	first, second := <-results, <-results
	if !((first == http.StatusOK && second == http.StatusConflict) || (second == http.StatusOK && first == http.StatusConflict)) {
		t.Fatalf("concurrent previews returned %d and %d; want one success and one conflict", first, second)
	}
}

func TestOccurrenceLifecycleCancellationPreservesPrivateState(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Cancellation boundary", 10)
	eventID := mustString(t, event, "id")
	ticket := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/test-ticket", map[string]any{
		"email": "guest@example.test", "displayName": "Guest",
	}, http.StatusOK)
	path := "/api/events/" + eventID + "/occurrences"
	created := postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
		"name": "Cancellation boundary", "startsAt": "2026-10-01T20:00:00Z",
	}, http.StatusOK)
	// Compare complete private rows and queue count; a public status edit is
	// neither an admission cancellation, refund, nor a claim that notices sent.
	snapshot := func() string {
		var data string
		if err := fx.app.db.QueryRow(t.Context(), `select json_build_object(
			'event', (select row_to_json(e) from events e where id=$1),
			'ticket', (select row_to_json(t) from tickets t where id=$2),
			'outboxCount', (select count(*) from email_outbox))::text`, eventID, mustString(t, ticket.JSON, "id")).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	path += "/" + mustString(t, created.JSON, "id")
	cancelled := patchJSON(t, fx.app, fx.ownerCookie, path, map[string]any{"status": "cancelled"}, http.StatusOK)
	if field(t, cancelled.JSON, "status") != "cancelled" || snapshot() != before {
		t.Fatal("public cancellation mutated private event, ticket or email queue")
	}
	// Editing dates on a cancelled listing must not implicitly reactivate it.
	updated := patchJSON(t, fx.app, fx.ownerCookie, path, map[string]any{"startsAt": "2026-10-02T20:00:00Z"}, http.StatusOK)
	if field(t, updated.JSON, "status") != "cancelled" || snapshot() != before {
		t.Fatal("date correction reactivated cancellation or changed private state")
	}
}

func TestOccurrenceLifecycleRejectsStalePreview(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Lifecycle preview", 10)
	eventID := mustString(t, event, "id")
	created := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"name": "Public occurrence", "startsAt": "2026-11-01T01:30:00-05:00", "timezone": "America/Chicago",
	}, http.StatusOK)
	id := mustString(t, created.JSON, "id")
	path := "/api/events/" + eventID + "/occurrences/" + id
	// A remote record observation changed after preview. CID is opaque here;
	// repository publication and CID syntax admission are separate boundaries.
	if _, err := fx.app.db.Exec(t.Context(), `update event_occurrences set public_cid='current-cid' where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	patchJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
		"status": "cancelled", "expectedPublicCid": "old-cid",
	}, http.StatusConflict)
	updated := patchJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
		"startsAt": "2026-11-01T01:30:00-06:00", "expectedPublicCid": "current-cid",
		"expectedUpdatedAt": mustString(t, created.JSON, "updatedAt"),
	}, http.StatusOK)
	if field(t, updated.JSON, "status") != "rescheduled" || field(t, updated.JSON, "startsAt") != "2026-11-01T07:30:00Z" {
		t.Fatalf("fall-back reschedule lost the selected instant: %#v", updated.JSON)
	}
	patchJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
		"status": "cancelled", "expectedUpdatedAt": mustString(t, created.JSON, "updatedAt"),
	}, http.StatusConflict)
	var status string
	if err := fx.app.db.QueryRow(t.Context(), `select status from event_occurrences where id=$1`, id).Scan(&status); err != nil || status != "rescheduled" {
		t.Fatalf("stale cancellation changed occurrence: status=%q err=%v", status, err)
	}
}

func TestOccurrenceLifecycleValidatesSchedule(t *testing.T) {
	fx := newCulturalFixture(t)
	event := createEvent(t, fx, "Schedule validation", 10)
	path := "/api/events/" + mustString(t, event, "id") + "/occurrences"
	for _, zone := range []string{"Not/A_Zone", "Local", "../UTC"} {
		postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
			"name": "Invalid zone", "startsAt": "2026-03-08T01:30:00-06:00", "timezone": zone,
		}, http.StatusBadRequest)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
		"name": "Invalid interval", "startsAt": "2026-03-08T01:30:00-06:00", "endsAt": "2026-03-08T01:00:00-06:00",
	}, http.StatusBadRequest)
	created := postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{
		"name": "Spring transition", "startsAt": "2026-03-08T01:30:00-06:00",
		"endsAt": "2026-03-08T03:30:00-05:00", "timezone": "America/Chicago",
	}, http.StatusOK)
	start, _ := time.Parse(time.RFC3339, mustString(t, created.JSON, "startsAt"))
	end, _ := time.Parse(time.RFC3339, mustString(t, created.JSON, "endsAt"))
	if end.Sub(start) != time.Hour {
		t.Fatal("spring transition must preserve elapsed time")
	}
	path += "/" + mustString(t, created.JSON, "id")
	for _, patch := range []map[string]any{
		{"timezone": "Not/A_Zone"}, {"startsAt": "2026-03-08T09:00:00Z"},
		{"endsAt": "2026-03-08T07:30:00Z"}, {"expectedUpdatedAt": "not-a-time", "status": "cancelled"},
	} {
		patchJSON(t, fx.app, fx.ownerCookie, path, patch, http.StatusBadRequest)
	}
}
