package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func venueAccessFixture(t *testing.T) (lifecycleFixture, string, string) {
	t.Helper()
	fx := newLifecycleFixture(t)
	place := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places", map[string]any{"name": "Synthetic access venue", "streetAddress": "PRIVATE_STREET", "accessNotes": "LEGACY_NOTE_NOT_AN_ASSERTION"}, 200)
	id := mustString(t, place.JSON, "id")
	return fx, id, "/api/workspaces/" + fx.workspaceID + "/places/" + id + "/access-info"
}
func TestVenueAccessScopePrivacyAndNoEventInheritance(t *testing.T) {
	fx, placeID, path := venueAccessFixture(t)
	getJSON(t, fx.app, nil, path, 401)
	getJSON(t, fx.app, fx.memberCookie, path, 403)
	postJSON(t, fx.app, fx.memberCookie, path+"/entry", accessPayload(0), 403)
	defaults := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
	if defaults["eventId"] != nil || defaults["placeId"] != placeID || len(defaults["entries"].([]any)) != 6 {
		t.Fatal("scope mismatch")
	}
	for _, e := range defaults["entries"].([]any) {
		x := mustObject(t, e)
		if x["value"] != "unknown" || x["scope"] != "venue" {
			t.Fatal("legacy venue note promoted to assertion")
		}
	}
	req := accessPayload(0)
	req["sourceKind"] = "venue_observation"
	record := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path+"/entry", req, 201).JSON)
	if record["scope"] != "venue" || record["sourceKind"] != "venue_observation" {
		t.Fatal("venue observation mislabeled")
	}
	bad := accessPayload(0)
	bad["sourceKind"] = "event_observation"
	postJSON(t, fx.app, fx.ownerCookie, path+"/bathrooms", bad, 400)
	eventID := mustString(t, createEvent(t, fx, "Venue-linked synthetic event", 10), "id")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{"name": "Linked venue occurrence", "startsAt": "2026-12-01T12:00:00Z", "timezone": "UTC", "placeId": placeID}, 200)
	worksheet := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/access-info", 200).JSON)
	for _, e := range worksheet["entries"].([]any) {
		x := mustObject(t, e)
		if x["value"] != "unknown" || x["scope"] != "event" {
			t.Fatal("venue assertion automatically inherited")
		}
	}
	bad["sourceKind"] = "venue_observation"
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/access-info/entry", bad, 400)
	// The public cultural place projection is a separate allowlisted serializer.
	place, err := fx.app.loadCulturalPlace(t.Context(), placeID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(buildPublicPlaceRecord(place))
	for _, private := range []string{"PRIVATE_ACCESS_SOURCE", "LEGACY_NOTE_NOT_AN_ASSERTION", "PRIVATE_STREET", "sourceKind", "accessRevisions"} {
		if strings.Contains(string(body), private) {
			t.Fatalf("public place leaked %s", private)
		}
	}
	other := newLifecycleFixture(t, fx.app)
	getJSON(t, fx.app, other.ownerCookie, path, 403)
	getJSON(t, fx.app, other.ownerCookie, "/api/workspaces/"+other.workspaceID+"/places/"+placeID+"/access-info", 404)
}
func TestVenueAccessReplayCorrectionAndCrossScopeKey(t *testing.T) {
	fx, _, path := venueAccessFixture(t)
	req := accessPayload(0)
	req["expiresAt"] = "2026-02-01T00:00:00Z"
	first := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path+"/entry", req, 201).JSON)
	if first["effectiveValue"] != "unknown" || first["needsReview"] != true {
		t.Fatal("venue expiry ignored")
	}
	next := accessPayload(1)
	next["value"] = "no"
	next["sourceKind"] = "venue_observation"
	postJSON(t, fx.app, fx.ownerCookie, path+"/entry", next, 201)
	replay := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path+"/entry", req, 200).JSON)
	if replay["id"] != first["id"] {
		t.Fatal("venue replay changed identity")
	}
	eventID := mustString(t, createEvent(t, fx, "Other scope", 10), "id")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/access-info/entry", req, 409)
	postJSON(t, fx.app, fx.ownerCookie, path+"/entry", accessPayload(1), 409)
	history := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path+"/entry/history", 200).JSON)
	if len(history["revisions"].([]any)) != 2 {
		t.Fatal("venue history lost")
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=now() where workspace_id=$1 and person_id=$2`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.ownerCookie, path, 403)
	getJSON(t, fx.app, fx.ownerCookie, path+"/entry/history", 403)
	postJSON(t, fx.app, fx.ownerCookie, path+"/entry", req, 403)
}
func TestVenueAccessConcurrentReplayAndAuditRollback(t *testing.T) {
	fx, _, path := venueAccessFixture(t)
	req := accessPayload(0)
	if _, err := fx.app.db.Exec(t.Context(), `alter table audit_entries add constraint synthetic_venue_access_audit_failure check(action<>'place_access.revised')`); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path+"/entry", req, 500)
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_access_revisions`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed venue audit left revision")
	}
	if _, err := fx.app.db.Exec(t.Context(), `alter table audit_entries drop constraint synthetic_venue_access_audit_failure`); err != nil {
		t.Fatal(err)
	}
	results := make(chan int, 2)
	for range 2 {
		go func() { results <- accessRequestStatus(fx, path+"/entry", req) }()
	}
	a, b := <-results, <-results
	if !((a == 200 && b == 201) || (a == 201 && b == 200)) {
		t.Fatalf("venue replay %d/%d", a, b)
	}
	one, two := accessPayload(1), accessPayload(1)
	two["value"] = "no"
	go func() { results <- accessRequestStatus(fx, path+"/entry", one) }()
	go func() { results <- accessRequestStatus(fx, path+"/entry", two) }()
	a, b = <-results, <-results
	if !((a == 201 && b == 409) || (a == 409 && b == 201)) {
		t.Fatalf("venue correction %d/%d", a, b)
	}
}
func TestVenueAccessMigrationPreservesEventRevisions(t *testing.T) {
	fx, eventID, path := accessFixture(t)
	// Reconstruct schema 28 in this disposable database, preserving existing IDs.
	if _, err := fx.app.db.Exec(t.Context(), `drop table venue_access_place_requests;alter table event_access_revisions drop constraint access_revision_source_scope;alter table event_access_revisions drop constraint access_revision_source_kind;alter table event_access_revisions add constraint event_access_revisions_source_kind_check check(source_kind in ('unknown','organizer_assertion','event_observation','external_reference'));alter table event_access_revisions drop constraint access_revision_one_scope;alter table event_access_revisions drop column place_id;alter table event_access_revisions alter column event_id set not null;delete from schema_migrations where version=29`); err != nil {
		t.Fatal(err)
	}
	// Use the actual v28 table shape rather than calling a schema29-only handler.
	key := uuid.NewString()
	var id string
	if err := fx.app.db.QueryRow(t.Context(), `insert into event_access_revisions(event_id,topic,revision,value,source_kind,correction_reason,recorded_by_person_id,request_key,request_fingerprint) values($1,'entry',1,'unknown','unknown','Preserved historical unknown',$2,$3,$4) returning id`, eventID, ownerPersonID(t, fx), key, strings.Repeat("a", 64)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	var before string
	if err := fx.app.db.QueryRow(t.Context(), `select to_jsonb(r)::text from event_access_revisions r where id=$1`, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := RunMigrations(t.Context(), fx.app.db); err != nil {
			t.Fatal(err)
		}
	}
	var after string
	if err := fx.app.db.QueryRow(t.Context(), `select (to_jsonb(r)-'place_id')::text from event_access_revisions r where id=$1`, id).Scan(&after); err != nil || before != after {
		t.Fatal("event revision changed on upgrade")
	}
	current := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
	entry := mustObject(t, current["entries"].([]any)[0])
	if entry["id"] != id || entry["scope"] != "event" {
		t.Fatal("upgraded event history lost")
	}
}

func TestVenueAccessReferenceCreationReplayPrivacyAndRollback(t *testing.T) {
	fx := newLifecycleFixture(t)
	path := "/api/workspaces/" + fx.workspaceID + "/venue-access"
	req := map[string]any{"name": " Synthetic named venue ", "requestKey": uuid.NewString()}
	getJSON(t, fx.app, nil, path, 401)
	getJSON(t, fx.app, fx.memberCookie, path, 403)
	postJSON(t, fx.app, fx.memberCookie, path, req, 403)
	postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{"name": "Venue"}, 400)
	first := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path, req, 201).JSON)
	replay := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, path, req, 200).JSON)
	if first["id"] != replay["id"] || first["name"] != "Synthetic named venue" {
		t.Fatal("creation replay changed venue")
	}
	req["name"] = "Different venue"
	postJSON(t, fx.app, fx.ownerCookie, path, req, 409)
	index := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
	if len(index["places"].([]any)) != 1 {
		t.Fatal("replay duplicated venue")
	}
	row := mustObject(t, index["places"].([]any)[0])
	if len(row) != 2 {
		t.Fatal("index leaked protected fields")
	}
	other := newLifecycleFixture(t, fx.app)
	postJSON(t, fx.app, other.ownerCookie, "/api/workspaces/"+other.workspaceID+"/venue-access", req, 409)
	if _, err := fx.app.db.Exec(t.Context(), `alter table audit_entries add constraint synthetic_reference_audit_failure check(action<>'place_access.reference_created') not valid`); err != nil {
		t.Fatal(err)
	}
	postJSON(t, fx.app, fx.ownerCookie, path, map[string]any{"name": "Rolled back venue", "requestKey": uuid.NewString()}, 500)
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from cultural_places where workspace_id=$1`, fx.workspaceID).Scan(&count); err != nil || count != 1 {
		t.Fatal("audit failure left venue reference")
	}
	if _, err := fx.app.db.Exec(t.Context(), `alter table audit_entries drop constraint synthetic_reference_audit_failure`); err != nil {
		t.Fatal(err)
	}
	concurrent := map[string]any{"name": "Concurrent venue", "requestKey": uuid.NewString()}
	results := make(chan int, 2)
	for range 2 {
		go func() { results <- accessRequestStatus(fx, path, concurrent) }()
	}
	a, b := <-results, <-results
	if !((a == 201 && b == 200) || (a == 200 && b == 201)) {
		t.Fatalf("reference replay %d/%d", a, b)
	}
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=now() where workspace_id=$1 and person_id=$2`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.ownerCookie, path, 403)
	postJSON(t, fx.app, fx.ownerCookie, path, concurrent, 403)
}
func TestVenueAccessReferencePagination(t *testing.T) {
	fx := newLifecycleFixture(t)
	path := "/api/workspaces/" + fx.workspaceID + "/venue-access"
	if _, err := fx.app.db.Exec(t.Context(), `insert into cultural_places(workspace_id,name,created_by_person_id) select $1,'Synthetic venue '||n,$2 from generate_series(1,103) n`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
		t.Fatal(err)
	}
	first := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
	if len(first["places"].([]any)) != 100 {
		t.Fatal("unbounded venue index")
	}
	next := mustString(t, first, "nextAfter")
	second := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path+"?after="+next, 200).JSON)
	if len(second["places"].([]any)) != 3 || second["nextAfter"] != nil {
		t.Fatal("venue pagination omitted records")
	}
	seen := map[string]bool{}
	for _, page := range []map[string]any{first, second} {
		for _, item := range page["places"].([]any) {
			id := mustString(t, item, "id")
			if seen[id] {
				t.Fatal("duplicate cursor entry")
			}
			seen[id] = true
		}
	}
	getJSON(t, fx.app, fx.ownerCookie, path+"?after=invalid", 400)
}

func TestVenueAccessRechecksOwnerAfterPlaceLock(t *testing.T) {
	fx, id, path := venueAccessFixture(t)
	tx, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	if _, err = tx.Exec(t.Context(), `select id from cultural_places where id=$1 for update`, id); err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() { result <- accessRequestStatus(fx, path+"/entry", accessPayload(0)) }()
	deadline := time.Now().Add(10 * time.Second)
	waiting := false
	for time.Now().Before(deadline) {
		var n int
		if err = fx.app.db.QueryRow(t.Context(), `select count(*) from pg_stat_activity where datname=current_database() and wait_event_type='Lock' and query like '%select id from cultural_places%for update%'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			waiting = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("request did not wait on venue lock")
	}
	if _, err = fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=now() where workspace_id=$1 and person_id=$2`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != 403 {
			t.Fatalf("stale owner admitted %d", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("request did not finish")
	}
	var n int
	if err = fx.app.db.QueryRow(t.Context(), `select count(*) from event_access_revisions`).Scan(&n); err != nil || n != 0 {
		t.Fatal("revoked owner wrote revision")
	}
}
