package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func comparisonFixture(t *testing.T) (lifecycleFixture, string, string, string) {
	t.Helper()
	fx, placeID, venuePath := venueAccessFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Synthetic comparison event", 10), "id")
	occurrence := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{"name": "Synthetic linked occurrence", "startsAt": "2026-12-01T12:00:00Z", "timezone": "UTC", "placeId": placeID}, 200)
	occurrenceID := mustString(t, occurrence.JSON, "id")
	req := accessPayload(0)
	req["sourceKind"] = "venue_observation"
	postJSON(t, fx.app, fx.ownerCookie, venuePath+"/entry", req, 201)
	return fx, eventID, occurrenceID, "/api/events/" + eventID + "/access-info/occurrences/" + occurrenceID + "/comparison"
}
func TestOccurrenceAccessComparisonScopeUnknownExpiryAndPrivacy(t *testing.T) {
	fx, eventID, occurrenceID, path := comparisonFixture(t)
	index := "/api/events/" + eventID + "/access-info/occurrences"
	getJSON(t, fx.app, nil, path, 401)
	getJSON(t, fx.app, fx.memberCookie, path, 403)
	getJSON(t, fx.app, fx.memberCookie, index, 403)
	out := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
	event := mustObject(t, out["event"])
	venue := mustObject(t, out["venue"])
	if event["evaluatedAt"] != out["evaluatedAt"] || venue["evaluatedAt"] != out["evaluatedAt"] {
		t.Fatal("comparison uses different expiry clocks")
	}
	if mustObject(t, event["entries"].([]any)[0])["value"] != "unknown" || mustObject(t, venue["entries"].([]any)[0])["value"] != "yes" {
		t.Fatal("venue assertion inherited into event")
	}
	req := accessPayload(0)
	req["value"] = "no"
	req["sourceKind"] = "event_observation"
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/access-info/entry", req, 201)
	out = mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
	if mustObject(t, mustObject(t, out["event"])["entries"].([]any)[0])["value"] != "no" {
		t.Fatal("event observation omitted")
	}
	body, _ := json.Marshal(out)
	for _, field := range []string{"PRIVATE_STREET", "LEGACY_NOTE_NOT_AN_ASSERTION", "recordedByPersonId", "requestKey", "requestFingerprint", "createdByPersonId", "publicCid"} {
		if strings.Contains(string(body), field) {
			t.Fatalf("comparison leaked %s", field)
		}
	}
	list := getJSON(t, fx.app, fx.ownerCookie, index, 200).JSON.([]any)
	if len(list) != 1 || mustString(t, list[0], "id") != occurrenceID || len(mustObject(t, list[0])) != 6 {
		t.Fatal("occurrence options are not minimal")
	}
	otherID := mustString(t, createEvent(t, fx, "Different event", 10), "id")
	getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+otherID+"/access-info/occurrences/"+occurrenceID+"/comparison", 404)
	getJSON(t, fx.app, fx.ownerCookie, index+"/invalid/comparison", 400)
	other := newLifecycleFixture(t, fx.app)
	getJSON(t, fx.app, other.ownerCookie, path, 403)
	expired := accessPayload(1)
	expired["sourceKind"] = "venue_observation"
	expired["expiresAt"] = "2026-02-01T00:00:00Z"
	placeID := mustString(t, venue, "placeId")
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/places/"+placeID+"/access-info/entry", expired, 201)
	out = mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
	if mustObject(t, mustObject(t, out["venue"])["entries"].([]any)[0])["effectiveValue"] != "unknown" {
		t.Fatal("expired venue assertion appears current")
	}
	unlinked := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences", map[string]any{"name": "No venue", "startsAt": "2026-12-02T12:00:00Z", "timezone": "UTC"}, 200)
	none := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, index+"/"+mustString(t, unlinked.JSON, "id")+"/comparison", 200).JSON)
	if none["venue"] != nil {
		t.Fatal("unlinked occurrence inferred venue")
	}
}
func TestOccurrenceAccessComparisonSnapshotAndOwnerRecheck(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		t.Run(map[bool]string{false: "snapshot", true: "revocation"}[revoke], func(t *testing.T) {
			fx, eventID, occurrenceID, path := comparisonFixture(t)
			lock, err := fx.app.db.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Rollback(t.Context())
			if _, err = lock.Exec(t.Context(), `lock table event_access_revisions in access exclusive mode`); err != nil {
				t.Fatal(err)
			}
			results := make(chan *httptest.ResponseRecorder, 1)
			go func() {
				r := httptest.NewRequest(http.MethodGet, path, nil)
				r.AddCookie(fx.ownerCookie)
				w := httptest.NewRecorder()
				fx.app.Handler().ServeHTTP(w, r)
				results <- w
			}()
			waiting := false
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				var n int
				if err = fx.app.db.QueryRow(t.Context(), `select count(*) from pg_stat_activity where datname=current_database() and wait_event_type='Lock' and query like '%from event_access_revisions%'`).Scan(&n); err != nil {
					t.Fatal(err)
				}
				if n > 0 {
					waiting = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("comparison did not reach snapshot worksheet read")
			}
			if revoke {
				if _, err = fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=now() where workspace_id=$1 and person_id=$2`, fx.workspaceID, ownerPersonID(t, fx)); err != nil {
					t.Fatal(err)
				}
			} else {
				patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/occurrences/"+occurrenceID, map[string]any{"clearPlace": true, "name": "Changed occurrence"}, 200)
				if _, err = fx.app.db.Exec(t.Context(), `update cultural_places set name='Changed venue' where workspace_id=$1`, fx.workspaceID); err != nil {
					t.Fatal(err)
				}
			}
			if err = lock.Commit(t.Context()); err != nil {
				t.Fatal(err)
			}
			select {
			case w := <-results:
				if w.Header().Get("Cache-Control") != "private, no-store" {
					t.Fatal("comparison missing private cache boundary")
				}
				if revoke {
					if w.Code != 403 {
						t.Fatalf("revoked owner received %d", w.Code)
					}
					if strings.Contains(w.Body.String(), "Synthetic") {
						t.Fatal("private comparison returned after revocation")
					}
					getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/access-info/occurrences", 403)
				} else {
					if w.Code != 200 {
						t.Fatalf("comparison status %d: %s", w.Code, w.Body.String())
					}
					var out map[string]any
					if err = json.Unmarshal(w.Body.Bytes(), &out); err != nil {
						t.Fatal(err)
					}
					if mustString(t, out["occurrence"], "name") != "Synthetic linked occurrence" || mustString(t, out["venue"], "placeName") != "Synthetic access venue" {
						t.Fatal("comparison mixed snapshots")
					}
					next := mustObject(t, getJSON(t, fx.app, fx.ownerCookie, path, 200).JSON)
					if mustString(t, next["occurrence"], "name") != "Changed occurrence" || next["venue"] != nil {
						t.Fatal("refresh retained stale occurrence")
					}
				}
			case <-time.After(10 * time.Second):
				t.Fatal("comparison did not finish")
			}
		})
	}
}
