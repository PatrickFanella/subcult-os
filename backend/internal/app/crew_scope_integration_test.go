package app

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// coordinatorOnlyReadPaths are reads that hold planning or other people's
// data. Owners and organizers keep them; crew, door and finance do not.
func coordinatorOnlyReadPaths(fx claimFixture) []string {
	event := "/api/events/" + fx.eventID
	workspace := "/api/workspaces/" + fx.workspaceID
	return []string{
		event + "/role-applications", event + "/participants", event + "/notifications", event + "/reminders",
		event + "/public-links", event + "/occurrences",
		workspace + "/contacts", workspace + "/reminders", workspace + "/archives", workspace + "/event-templates",
		workspace + "/places", workspace + "/profiles",
	}
}

func personIDByEmail(t *testing.T, fx claimFixture, email string) string {
	t.Helper()
	var id string
	if err := fx.app.db.QueryRow(t.Context(), `select id from people where email = $1`, email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCrewSeesOnlyTheEventsTheyWork(t *testing.T) {
	fx := newClaimFixture(t)
	crewEmail, ownerEmail := fx.email("member"), fx.email("owner")
	crewID, ownerID := personIDByEmail(t, fx, crewEmail), personIDByEmail(t, fx, ownerEmail)

	organizerEmail := fx.email("organizer")
	invite := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": organizerEmail, "role": "crew"}, http.StatusOK)
	organizerCookie := signupAndVerifyCookie(t, fx.app, organizerEmail, "Organizer")
	postJSON(t, fx.app, organizerCookie, "/api/invitations/"+mustString(t, invite.JSON, "token")+"/accept", map[string]any{}, http.StatusOK)
	if _, err := fx.app.db.Exec(t.Context(), `update workspace_members set role = 'organizer' where workspace_id = $1 and person_id = $2`, fx.workspaceID, personIDByEmail(t, fx, organizerEmail)); err != nil {
		t.Fatal(err)
	}

	starts := claimClock.Add(24 * time.Hour)
	shift := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+fx.eventID+"/staffing", map[string]any{
		"title": "Door shift", "kind": "shift", "notes": "Cash box code 4471", "participantRequirements": "Bring a torch.",
		"startsAt": starts.Format(time.RFC3339), "endsAt": starts.Add(3 * time.Hour).Format(time.RFC3339),
	}, http.StatusOK)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+fx.eventID+"/staffing/"+mustString(t, shift.JSON, "id"), map[string]any{"assignedPersonId": crewID}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/contacts", map[string]any{"displayName": "Venue manager", "email": "venue@example.test"}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{"title": "Crew task", "eventId": fx.eventID, "ownerPersonId": crewID}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/commitments", map[string]any{"title": "Owner task", "eventId": fx.eventID, "ownerPersonId": ownerID}, http.StatusOK)

	// Crew keep the events they work, the run of show and the roles.
	for _, path := range []string{"/api/workspaces/" + fx.workspaceID + "/events", "/api/events/" + fx.eventID, "/api/events/" + fx.eventID + "/roles"} {
		getJSON(t, fx.app, fx.memberCookie, path, http.StatusOK)
	}
	staffing := getJSON(t, fx.app, fx.memberCookie, "/api/events/"+fx.eventID+"/staffing", http.StatusOK)
	if !strings.Contains(staffing.Body, "Door shift") || !strings.Contains(staffing.Body, "Bring a torch.") || !strings.Contains(staffing.Body, mustString(t, shift.JSON, "id")) {
		t.Fatalf("crew run of show is missing the shift: %s", staffing.Body)
	}
	for _, private := range []string{"Cash box code 4471", crewEmail, ownerEmail} {
		if strings.Contains(staffing.Body, private) {
			t.Fatalf("crew run of show exposed %q: %s", private, staffing.Body)
		}
	}
	if owner := getJSON(t, fx.app, fx.ownerCookie, "/api/events/"+fx.eventID+"/staffing", http.StatusOK); !strings.Contains(owner.Body, "Cash box code 4471") {
		t.Fatalf("owner lost the operator notes: %s", owner.Body)
	}

	// Commitments: crew see only their own; coordinators see all.
	for _, path := range []string{"/api/workspaces/" + fx.workspaceID + "/commitments", "/api/events/" + fx.eventID + "/commitments"} {
		crew := getJSON(t, fx.app, fx.memberCookie, path, http.StatusOK)
		if !strings.Contains(crew.Body, "Crew task") || strings.Contains(crew.Body, "Owner task") {
			t.Fatalf("crew %s = %s", path, crew.Body)
		}
		organizer := getJSON(t, fx.app, organizerCookie, path, http.StatusOK)
		if !strings.Contains(organizer.Body, "Crew task") || !strings.Contains(organizer.Body, "Owner task") {
			t.Fatalf("organizer %s = %s", path, organizer.Body)
		}
	}

	// The workspace shows crew its name and their role, not the member list.
	workspace := getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID, http.StatusOK)
	if members, _ := mustObject(t, workspace.JSON)["members"].([]any); len(members) != 0 || strings.Contains(workspace.Body, ownerEmail) {
		t.Fatalf("crew workspace detail = %s", workspace.Body)
	}
	organizerWorkspace := getJSON(t, fx.app, organizerCookie, "/api/workspaces/"+fx.workspaceID, http.StatusOK)
	if members, _ := mustObject(t, organizerWorkspace.JSON)["members"].([]any); len(members) < 3 {
		t.Fatalf("organizer lost the member list: %s", organizerWorkspace.Body)
	}

	// Planning and other people's data stay with coordinators.
	for _, path := range coordinatorOnlyReadPaths(fx) {
		if code, body := rawJSON(fx.app, fx.memberCookie, http.MethodGet, path, nil); code != http.StatusForbidden {
			t.Errorf("crew GET %s = %d %s", path, code, body)
		}
		if code, body := rawJSON(fx.app, organizerCookie, http.MethodGet, path, nil); code != http.StatusOK {
			t.Errorf("organizer GET %s = %d %s", path, code, body)
		}
	}
	if code, _ := rawJSON(fx.app, fx.memberCookie, http.MethodPost, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Crew event", "startsAt": starts.Format(time.RFC3339), "ticketAllocation": 10}); code != http.StatusForbidden {
		t.Errorf("crew created an event: %d", code)
	}
	if code, _ := rawJSON(fx.app, fx.memberCookie, http.MethodPatch, "/api/events/"+fx.eventID, map[string]any{"title": "Renamed by crew"}); code != http.StatusForbidden {
		t.Errorf("crew edited the event: %d", code)
	}
	draft := postJSON(t, fx.app, organizerCookie, "/api/workspaces/"+fx.workspaceID+"/events", map[string]any{"title": "Organizer draft", "publicDescription": "A draft.", "locationDisplay": "The Hall", "startsAt": starts.Format(time.RFC3339), "ticketAllocation": 10}, http.StatusOK)
	patchJSON(t, fx.app, organizerCookie, "/api/events/"+mustString(t, draft.JSON, "id"), map[string]any{"title": "Organizer draft, renamed"}, http.StatusOK)
	if code, _ := rawJSON(fx.app, fx.memberCookie, http.MethodPatch, "/api/events/"+mustString(t, draft.JSON, "id"), map[string]any{"title": "Renamed by crew"}); code != http.StatusForbidden {
		t.Errorf("crew edited a draft: %d", code)
	}
}
