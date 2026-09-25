package app

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func createLifecycleIntentOccurrence(t *testing.T, fx lifecycleFixture, eventID string) map[string]any {
	t.Helper()
	return mustObject(t, postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{
		"name": "Exact listing", "startsAt": "2026-11-07T02:30:00-06:00", "timezone": "America/Chicago",
	}, http.StatusOK).JSON)
}

func lifecycleIntentPayload(occurrence map[string]any) map[string]any {
	publicCID, _ := occurrence["publicCid"].(string)
	return map[string]any{"decisionKey": uuid.NewString(), "occurrenceId": occurrence["id"], "expectedUpdatedAt": occurrence["updatedAt"], "expectedPublicCid": publicCID, "kind": "cancellation", "reason": "weather", "actionKinds": []string{"operational_notice", "provider_ticket"}}
}

func TestLifecycleIntentHTTPIsOwnerOnlyFencedAndDraftOnly(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Intent event", 2), "id")
	occurrence := createLifecycleIntentOccurrence(t, fx, eventID)
	payload := lifecycleIntentPayload(occurrence)
	member := postJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/lifecycle-intents", payload, http.StatusForbidden)
	if member.JSON == nil {
		t.Fatal("member response missing")
	}
	other := newLifecycleFixture(t, fx.app)
	postJSON(t, fx.app, other.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", payload, http.StatusForbidden)
	stale := lifecycleIntentPayload(occurrence)
	stale["expectedUpdatedAt"] = "2020-01-01T00:00:00Z"
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", stale, http.StatusConflict)
	if _, err := fx.app.db.Exec(t.Context(), `update event_occurrences set public_cid='bafy-current' where id=$1`, occurrence["id"]); err != nil {
		t.Fatal(err)
	}
	staleCID := lifecycleIntentPayload(occurrence)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", staleCID, http.StatusConflict)
	occurrence["publicCid"] = "bafy-current"
	payload = lifecycleIntentPayload(occurrence)
	badAction := lifecycleIntentPayload(occurrence)
	badAction["actionKinds"] = []string{"https://provider.example/refund"}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", badAction, http.StatusBadRequest)
	created := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", payload, http.StatusCreated)
	if got := created.Header.Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control=%q", got)
	}
	intent := mustObject(t, created.JSON)
	if intent["status"] != "approved" || intent["occurrenceId"] != occurrence["id"] || intent["targetRevision"] != occurrence["updatedAt"] {
		t.Fatalf("unexpected intent: %#v", intent)
	}
	actions := intent["actions"].([]any)
	if len(actions) != 2 {
		t.Fatalf("actions=%#v", actions)
	}
	for _, raw := range actions {
		action := mustObject(t, raw)
		if action["status"] != "pending" || int(action["attemptCount"].(float64)) != 0 {
			t.Fatalf("unexpected action=%#v", action)
		}
	}
	updatedOccurrence := mustObject(t, patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrence["id"].(string), map[string]any{"name": "Edited after decision", "expectedUpdatedAt": occurrence["updatedAt"], "expectedPublicCid": occurrence["publicCid"]}, http.StatusOK).JSON)
	if updatedOccurrence["updatedAt"] == occurrence["updatedAt"] {
		t.Fatal("occurrence revision did not change")
	}
	replayed := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", payload, http.StatusOK).JSON)
	if replayed["id"] != intent["id"] {
		t.Fatalf("replay created a new intent: %#v", replayed)
	}
	listed := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", http.StatusOK).JSON.([]any)
	if len(listed) != 1 {
		t.Fatalf("listed=%#v", listed)
	}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", payload, http.StatusOK)
	var actionCount int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_lifecycle_actions where change_id=$1`, intent["id"]).Scan(&actionCount); err != nil || actionCount != 2 {
		t.Fatalf("action count=%d err=%v", actionCount, err)
	}
	superset := lifecycleIntentPayload(occurrence)
	superset["decisionKey"] = payload["decisionKey"]
	superset["actionKinds"] = []string{"operational_notice", "provider_ticket", "refund"}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", superset, http.StatusConflict)
	subset := lifecycleIntentPayload(occurrence)
	subset["decisionKey"] = payload["decisionKey"]
	subset["actionKinds"] = []string{"operational_notice"}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", subset, http.StatusConflict)
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_lifecycle_actions where change_id=$1`, intent["id"]).Scan(&actionCount); err != nil || actionCount != 2 {
		t.Fatalf("action count after conflicts=%d err=%v", actionCount, err)
	}
	if _, err := createLifecycleChange(t.Context(), fx.app.db, lifecycleChangeInput{WorkspaceID: fx.workspaceID, EventID: eventID, Kind: "reschedule", Reason: "legacy ledger", TargetRevision: "legacy-rev", DecisionSnapshot: json.RawMessage(`{"legacy":true,"nested":{"state":"kept"}}`), ApprovedByPersonID: ownerPersonID(t, fx)}); err != nil {
		t.Fatal(err)
	}
	legacyListed := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", http.StatusOK).JSON.([]any)
	if len(legacyListed) != 2 {
		t.Fatalf("expected generic migration-16 decision to remain listable: %#v", legacyListed)
	}
	conflictKey := lifecycleIntentPayload(occurrence)
	conflictKey["decisionKey"] = payload["decisionKey"]
	conflictKey["reason"] = "different reason"
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", conflictKey, http.StatusConflict)
	getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/lifecycle-intents", http.StatusForbidden)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=clock_timestamp() where workspace_id=$1 and person_id=$2`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", http.StatusForbidden)
}

func TestLifecycleIntentSupersedeOnlyStopsUnsentActions(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Supersede intent", 2), "id")
	occurrence := createLifecycleIntentOccurrence(t, fx, eventID)
	intent := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents", lifecycleIntentPayload(occurrence), http.StatusCreated).JSON)
	claim, err := claimLifecycleAction(t.Context(), fx.app.db)
	if err != nil || claim == nil {
		t.Fatalf("claim=%+v err=%v", claim, err)
	}
	superseded := mustObject(t, postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/lifecycle-intents/"+intent["id"].(string)+"/supersede", map[string]any{}, http.StatusOK).JSON)
	if superseded["status"] != "superseded" {
		t.Fatalf("superseded=%#v", superseded)
	}
	actions := superseded["actions"].([]any)
	statuses := map[string]string{}
	for _, raw := range actions {
		action := mustObject(t, raw)
		statuses[action["id"].(string)] = action["status"].(string)
	}
	if statuses[claim.ID] != "running" {
		t.Fatalf("running action was altered by supersession: %#v", statuses)
	}
	for id, status := range statuses {
		if id != claim.ID && status != "superseded" {
			t.Fatalf("unsent action %s status=%s", id, status)
		}
	}
}
