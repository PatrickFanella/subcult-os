package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func pollCreateBody() map[string]any {
	return map[string]any{"prompt": "Which call time works?", "options": []map[string]any{
		{"startsAt": "2026-11-01T06:30:00.123456Z", "endsAt": "2026-11-01T07:15:00.123456Z", "timezone": "America/Chicago"},
		{"startsAt": "2026-11-01T07:30:00.123456Z", "endsAt": "2026-11-01T08:15:00.123456Z", "timezone": "America/Chicago"},
	}}
}
func createAvailabilityPoll(t *testing.T, fx lifecycleFixture, eventID string, showNames bool) (string, string, availabilityPoll) {
	t.Helper()
	body := pollCreateBody()
	body["showNames"] = showNames
	result := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/polls", body, 201)
	u, err := url.Parse(mustString(t, result.JSON, "url"))
	if err != nil {
		t.Fatal(err)
	}
	id := mustString(t, result.JSON, "id")
	rec := availabilityRequest(fx.app, id, u.Fragment, "", nil)
	if rec.Code != 200 {
		t.Fatalf("read poll=%d %s", rec.Code, rec.Body.String())
	}
	var poll availabilityPoll
	if err := json.Unmarshal(rec.Body.Bytes(), &poll); err != nil {
		t.Fatal(err)
	}
	return id, u.Fragment, poll
}
func availabilityRequest(app *App, id, token, edit string, body any) *httptest.ResponseRecorder {
	method := http.MethodGet
	path := "/api/availability-polls/" + id
	var payload []byte
	if body != nil {
		method = http.MethodPost
		path += "/responses"
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(pollLinkHeader, token)
	req.Header.Set(pollEditHeader, edit)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	return rec
}
func assertAvailabilityStatus(t *testing.T, app *App, id, token, edit string, body any, status int) *httptest.ResponseRecorder {
	t.Helper()
	rec := availabilityRequest(app, id, token, edit, body)
	if rec.Code != status {
		t.Fatalf("poll status=%d want=%d body=%s", rec.Code, status, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("poll response can be cached")
	}
	return rec
}
func availabilityAnswers(poll availabilityPoll, answer string) map[string]string {
	out := map[string]string{}
	for _, o := range poll.Options {
		out[o.ID] = answer
	}
	return out
}

func TestAvailabilityPollPrivateThreeRespondentJourneyAndPick(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "PRIVATE EVENT SENTINEL", 10), "id")
	id, token, poll := createAvailabilityPoll(t, fx, eventID, false)
	edits := []string{strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)}
	for i, name := range []string{"Private Performer", "Private Crew", "Private Venue"} {
		assertAvailabilityStatus(t, fx.app, id, token, edits[i], availabilityResponse{DisplayName: name, PrivateNote: "PRIVATE NOTE SENTINEL", Answers: availabilityAnswers(poll, "yes")}, 200)
	}
	result := assertAvailabilityStatus(t, fx.app, id, token, "", nil, 200)
	for _, private := range []string{eventID, fx.workspaceID, "PRIVATE EVENT SENTINEL", "Private Performer", "Private Crew", "Private Venue", "PRIVATE NOTE SENTINEL", token, edits[0]} {
		if strings.Contains(result.Body.String(), private) {
			t.Fatalf("public poll leaked %q", private)
		}
	}
	if result.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Fatal("public poll is indexable")
	}
	var public availabilityPoll
	if err := json.Unmarshal(result.Body.Bytes(), &public); err != nil {
		t.Fatal(err)
	}
	if public.Totals[poll.Options[0].ID]["yes"] != 3 || public.Responses != nil {
		t.Fatalf("anonymous totals=%#v", public)
	}
	owner := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/polls", 200)
	if !strings.Contains(owner.Body, "Private Performer") || !strings.Contains(owner.Body, "PRIVATE NOTE SENTINEL") {
		t.Fatal("organizer cannot see full responses")
	}
	// Reopening with one's edit capability returns only one's name and note.
	own := assertAvailabilityStatus(t, fx.app, id, token, edits[0], nil, 200)
	if !strings.Contains(own.Body.String(), "Private Performer") || strings.Contains(own.Body.String(), "Private Crew") {
		t.Fatal("edit capability read another response")
	}
	assertAvailabilityStatus(t, fx.app, id, token, edits[0], availabilityResponse{DisplayName: "Updated Performer", Answers: availabilityAnswers(poll, "no")}, 200)
	// Replays update the same response rather than adding a fourth one.
	assertAvailabilityStatus(t, fx.app, id, token, edits[0], availabilityResponse{DisplayName: "Updated Performer", Answers: availabilityAnswers(poll, "no")}, 200)
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_availability_responses where poll_id=$1`, id).Scan(&count); err != nil || count != 3 {
		t.Fatalf("responses=%d err=%v", count, err)
	}
	chosen := poll.Options[1]
	updated := patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID, map[string]any{"pollOptionId": chosen.ID}, 200)
	if mustString(t, updated.JSON, "startsAt") != chosen.StartsAt || mustString(t, updated.JSON, "endsAt") != *chosen.EndsAt {
		t.Fatalf("selected times=%s", updated.Body)
	}
	assertAvailabilityStatus(t, fx.app, id, token, edits[2], availabilityResponse{DisplayName: "Late", Answers: availabilityAnswers(poll, "yes")}, 409)
	var status string
	var option string
	if err := fx.app.db.QueryRow(t.Context(), `select status,chosen_option_id from event_availability_polls where id=$1`, id).Scan(&status, &option); err != nil || status != "closed" || option != chosen.ID {
		t.Fatalf("choice=%s %s %v", status, option, err)
	}
	getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID, 200)
	publishEvent(t, fx, eventID)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID, map[string]any{"pollOptionId": poll.Options[0].ID}, 409)
	// Choices never create public occurrences, publish records or send mail.
	var occurrences int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_occurrences where event_id=$1`, eventID).Scan(&occurrences); err != nil || occurrences != 0 {
		t.Fatalf("poll silently dual-published: %d %v", occurrences, err)
	}
}

func TestAvailabilityPollRolesNamesBoundsRateCloseDelete(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Poll authority", 10), "id")
	path := "/api/events/" + eventID + "/polls"
	_, memberID := memberIdentity(t, fx)
	for _, role := range []string{roleDoor, roleFinance, roleCrew, roleMember, roleParticipant} {
		promoteFixtureMember(t, fx, role)
		postJSON(t, fx.app, fx.memberCookie, path, pollCreateBody(), 403)
		getJSON(t, fx.app, fx.memberCookie, path, 403)
	}
	promoteFixtureMember(t, fx, roleOrganizer)
	postJSON(t, fx.app, fx.memberCookie, path, pollCreateBody(), 201)
	getJSON(t, fx.app, fx.memberCookie, path, 200)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set revoked_at=now() where id=$1`, memberID); err != nil {
		t.Fatal(err)
	}
	getJSON(t, fx.app, fx.memberCookie, path, 403)
	postJSON(t, fx.app, fx.memberCookie, path, pollCreateBody(), 403)
	foreign := newLifecycleFixture(t, fx.app)
	getJSON(t, fx.app, foreign.ownerCookie, path, 403)
	id, token, poll := createAvailabilityPoll(t, fx, eventID, true)
	edit := strings.Repeat("d", 40)
	assertAvailabilityStatus(t, fx.app, id, "wrong-token-but-a-long-value-for-testing", "", nil, 404)
	assertAvailabilityStatus(t, fx.app, id, token, "short", availabilityResponse{}, 400)
	valid := availabilityResponse{DisplayName: "Visible Name", PrivateNote: "HIDDEN-NOTE", Answers: availabilityAnswers(poll, "yes")}
	assertAvailabilityStatus(t, fx.app, id, token, edit, valid, 200)
	rec := assertAvailabilityStatus(t, fx.app, id, token, "", nil, 200)
	if !strings.Contains(rec.Body.String(), "Visible Name") || strings.Contains(rec.Body.String(), "HIDDEN-NOTE") {
		t.Fatal("name setting disclosed private note")
	}
	for _, bad := range []availabilityResponse{{DisplayName: strings.Repeat("a", 81), Answers: valid.Answers}, {DisplayName: "Name", PrivateNote: strings.Repeat("x", 1001), Answers: valid.Answers}, {DisplayName: "Name", Answers: map[string]string{poll.Options[0].ID: "yes"}}, {DisplayName: "Name", Answers: availabilityAnswers(poll, "maybe")}} {
		assertAvailabilityStatus(t, fx.app, id, token, edit, bad, 400)
	}
	// A different valid edit token cannot overwrite the first response.
	assertAvailabilityStatus(t, fx.app, id, token, strings.Repeat("e", 40), availabilityResponse{DisplayName: "Other Name", Answers: availabilityAnswers(poll, "no")}, 200)
	own := assertAvailabilityStatus(t, fx.app, id, token, edit, nil, 200)
	if !strings.Contains(own.Body.String(), "Visible Name") {
		t.Fatal("another capability overwrote first response")
	}
	if _, err := fx.app.db.Exec(t.Context(), `update event_availability_polls set rate_count=30,rate_window=clock_timestamp() where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	assertAvailabilityStatus(t, fx.app, id, token, edit, valid, 429)
	if _, err := fx.app.db.Exec(t.Context(), `update event_availability_polls set rate_window=clock_timestamp()-interval '2 minutes' where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	assertAvailabilityStatus(t, fx.app, id, token, edit, valid, 200)
	postJSON(t, fx.app, fx.ownerCookie, path+"/"+id+"/close", map[string]any{}, 200)
	assertAvailabilityStatus(t, fx.app, id, token, edit, valid, 409)
	// Delete cascades through answers; event deletion has the same boundary.
	req := httptest.NewRequest(http.MethodDelete, path+"/"+id, nil)
	req.AddCookie(fx.ownerCookie)
	out := httptest.NewRecorder()
	fx.app.Handler().ServeHTTP(out, req)
	if out.Code != 200 {
		t.Fatalf("delete=%d %s", out.Code, out.Body.String())
	}
	assertAvailabilityStatus(t, fx.app, id, token, "", nil, 404)
	var responses, answers int
	if err := fx.app.db.QueryRow(t.Context(), `select (select count(*) from event_availability_responses where poll_id=$1),(select count(*) from event_availability_answers where poll_id=$1)`, id).Scan(&responses, &answers); err != nil || responses+answers != 0 {
		t.Fatal("deleted poll retained responses")
	}
	id, token, poll = createAvailabilityPoll(t, fx, eventID, false)
	assertAvailabilityStatus(t, fx.app, id, token, edit, availabilityResponse{DisplayName: "To delete", Answers: availabilityAnswers(poll, "yes")}, 200)
	if _, err := fx.app.db.Exec(t.Context(), `delete from events where id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_availability_responses where poll_id=$1`, id).Scan(&responses); err != nil || responses != 0 {
		t.Fatal("event deletion retained poll responses")
	}
}

func TestAvailabilityPollConcurrentEditsAndPublishedChoiceRace(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Poll race", 10), "id")
	id, token, poll := createAvailabilityPoll(t, fx, eventID, false)
	edit := strings.Repeat("f", 40)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for _, name := range []string{"First", "Second"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			results <- availabilityRequest(fx.app, id, token, edit, availabilityResponse{DisplayName: name, Answers: availabilityAnswers(poll, "yes")}).Code
		}(name)
	}
	wg.Wait()
	close(results)
	for status := range results {
		if status != 200 {
			t.Fatalf("concurrent answer=%d", status)
		}
	}
	var count int
	if err := fx.app.db.QueryRow(t.Context(), `select count(*) from event_availability_responses where poll_id=$1`, id).Scan(&count); err != nil || count != 1 {
		t.Fatal("concurrent replay duplicated response")
	}
	blocker, err := fx.app.db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(t.Context(), `select id from events where id=$1 for update`, eventID); err != nil {
		t.Fatal(err)
	}
	var blockerPID int
	if err := blocker.QueryRow(t.Context(), `select pg_backend_pid()`).Scan(&blockerPID); err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() {
		body, _ := json.Marshal(map[string]string{"pollOptionId": poll.Options[0].ID})
		req := httptest.NewRequest(http.MethodPatch, "/api/events/"+eventID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(fx.ownerCookie)
		out := httptest.NewRecorder()
		fx.app.Handler().ServeHTTP(out, req)
		result <- out.Code
	}()
	deadline := time.Now().Add(10 * time.Second)
	blocked := false
	for time.Now().Before(deadline) {
		var waiting int
		if err := fx.app.db.QueryRow(t.Context(), `select count(*) from pg_stat_activity where datname=current_database() and $1=any(pg_blocking_pids(pid))`, blockerPID).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			blocked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("poll choice did not lock event")
	}
	if _, err := blocker.Exec(t.Context(), `update events set status='published' where id=$1`, eventID); err != nil {
		t.Fatal(err)
	}
	if err := blocker.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != 409 {
			t.Fatalf("published choice=%d", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("poll choice stuck")
	}
}

func TestAvailabilityPollExpiryValidationAndScope(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Poll validation", 2), "id")
	id, token, poll := createAvailabilityPoll(t, fx, eventID, false)
	edit := strings.Repeat("g", 40)
	if _, err := fx.app.db.Exec(t.Context(), `update event_availability_polls set closes_at=clock_timestamp() where id=$1`, id); err != nil {
		t.Fatal(err)
	}
	assertAvailabilityStatus(t, fx.app, id, token, edit, availabilityResponse{DisplayName: "Late", Answers: availabilityAnswers(poll, "yes")}, 409)
	otherID := mustString(t, createEvent(t, fx, "Other event", 2), "id")
	patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+otherID, map[string]any{"pollOptionId": poll.Options[0].ID}, 404)
	body := pollCreateBody()
	body["options"] = []map[string]any{{"startsAt": "2026-11-01T06:30:00Z", "timezone": "Not/AZone"}, {"startsAt": "2026-11-02T06:30:00Z", "timezone": "UTC"}}
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/polls", body, 400)
	body = pollCreateBody()
	body["prompt"] = strings.Repeat("x", 501)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/polls", body, 400)
	publishEvent(t, fx, eventID)
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/polls", pollCreateBody(), 409)
}
