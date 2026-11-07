package app

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// participantOperatorReadPaths are the operator read routes a participant
// must be refused. Every one of them is open to crew and above.
func participantOperatorReadPaths(fx claimFixture) []string {
	event := "/api/events/" + fx.eventID
	workspace := "/api/workspaces/" + fx.workspaceID
	return []string{
		event, event + "/staffing", event + "/roles", event + "/role-applications", event + "/participants",
		event + "/commitments", event + "/notifications", event + "/reminders", event + "/report",
		event + "/settlement", event + "/finance-lines", event + "/door/tickets", event + "/public-links",
		event + "/occurrences", event + "/access-info", event + "/archive", event + "/lifecycle-intents",
		workspace + "/events", workspace + "/contacts", workspace + "/commitments", workspace + "/reminders",
		workspace + "/archives", workspace + "/event-templates", workspace + "/places", workspace + "/profiles",
		workspace + "/venue-access", workspace + "/announcements", workspace + "/consent-grants",
		workspace + "/delegations", workspace + "/backline-bindings", workspace + "/listmonk-bindings",
	}
}

func addParticipantTierMember(t *testing.T, fx claimFixture, prefix string) claimParticipant {
	t.Helper()
	email := fx.email(prefix)
	name := "Participant " + prefix
	invite := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": email, "role": "participant"}, http.StatusOK)
	if got := mustString(t, invite.JSON, "role"); got != "participant" {
		t.Fatalf("invitation role = %q", got)
	}
	cookie := signupAndVerifyCookie(t, fx.app, email, name)
	postJSON(t, fx.app, cookie, "/api/invitations/"+mustString(t, invite.JSON, "token")+"/accept", map[string]any{}, http.StatusOK)
	var personID, role string
	if err := fx.app.db.QueryRow(t.Context(), `
		select p.id, wm.role from people p join workspace_members wm on wm.person_id = p.id and wm.workspace_id = $2 where p.email = $1
	`, email, fx.workspaceID).Scan(&personID, &role); err != nil {
		t.Fatal(err)
	}
	if role != roleParticipant {
		t.Fatalf("accepted membership role = %q", role)
	}
	return claimParticipant{cookie: cookie, personID: personID, email: email, name: name}
}

func TestParticipantTierClaimsShiftsWithoutOperatorAccess(t *testing.T) {
	fx := newClaimFixture(t)
	participant := addParticipantTierMember(t, fx, "tier")
	applyForClaimRole(t, fx, participant, fx.roleID, "accepted")
	starts := claimClock.Add(48 * time.Hour)
	staffingID := createClaimShift(t, fx, "Bar back", starts, starts.Add(3*time.Hour), 2, 2)

	// The participant tier keeps its own participation surface.
	if _, ok := openShift(t, fx, participant, staffingID); !ok {
		t.Fatal("participant does not see the shift they are eligible for")
	}
	postJSON(t, fx.app, participant.cookie, claimPath(staffingID, "claim"), nil, http.StatusOK)
	portal := getParticipantPortal(t, fx.app, participant.cookie, http.StatusOK)
	if len(portal.Assignments) != 1 || portal.Assignments[0].StaffingItemID != staffingID {
		t.Fatalf("participant portal = %#v", portal.Assignments)
	}
	if code, calendar := rawJSON(fx.app, participant.cookie, http.MethodGet, "/api/me/participant-portal/shifts.ics", nil); code != http.StatusOK || !strings.Contains(calendar, "Bar back") {
		t.Fatalf("participant calendar = %d %s", code, calendar)
	}

	// The workspace itself shows only its name and the participant's role.
	for _, path := range []string{"/api/workspaces/" + fx.workspaceID, "/api/workspaces/current"} {
		response := getJSON(t, fx.app, participant.cookie, path, http.StatusOK)
		workspace := mustObject(t, response.JSON)
		members, _ := workspace["members"].([]any)
		invitations, _ := workspace["invitations"].([]any)
		if workspace["role"] != roleParticipant || len(members) != 0 || len(invitations) != 0 {
			t.Fatalf("%s for participant = %s", path, response.Body)
		}
		if strings.Contains(response.Body, fx.email("owner")) {
			t.Fatalf("%s exposed the owner's email: %s", path, response.Body)
		}
	}

	// Every operator read is refused, while crew still has it.
	for _, path := range participantOperatorReadPaths(fx) {
		code, body := rawJSON(fx.app, participant.cookie, http.MethodGet, path, nil)
		if code < 400 {
			t.Errorf("participant GET %s = %d %s", path, code, body)
		}
		for _, private := range []string{"operator-only note", fx.email("owner")} {
			if strings.Contains(body, private) {
				t.Errorf("participant GET %s exposed %q", path, private)
			}
		}
	}
	for _, path := range []string{"/api/events/" + fx.eventID + "/staffing", "/api/workspaces/" + fx.workspaceID + "/events"} {
		getJSON(t, fx.app, fx.memberCookie, path, http.StatusOK)
	}

	// Participants cannot write operator data either.
	if code, body := rawJSON(fx.app, participant.cookie, http.MethodPost, "/api/workspaces/"+fx.workspaceID+"/contacts", map[string]any{"displayName": "Leak"}); code < 400 {
		t.Fatalf("participant created a contact: %d %s", code, body)
	}
}

func TestParticipantTierInvitationsAndRoleChanges(t *testing.T) {
	fx := newClaimFixture(t)
	for _, role := range []string{"owner", "organizer", "door", "finance", "admin"} {
		postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": fx.email("bad-" + role), "role": role}, http.StatusBadRequest)
	}
	crewInvite := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": fx.email("crew-tier"), "role": "crew"}, http.StatusOK)
	if mustString(t, crewInvite.JSON, "role") != roleCrew {
		t.Fatalf("crew invitation = %s", crewInvite.Body)
	}
	defaultInvite := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": fx.email("default-tier")}, http.StatusOK)
	if mustString(t, defaultInvite.JSON, "role") != roleMember {
		t.Fatalf("default invitation = %s", defaultInvite.Body)
	}

	// An owner moves a participant up to crew and back down.
	participant := addParticipantTierMember(t, fx, "moves")
	var memberID string
	if err := fx.app.db.QueryRow(t.Context(), `select id from workspace_members where workspace_id = $1 and person_id = $2`, fx.workspaceID, participant.personID).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	eventsPath := "/api/workspaces/" + fx.workspaceID + "/events"
	getJSON(t, fx.app, participant.cookie, eventsPath, http.StatusForbidden)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberID, map[string]any{"role": "crew"}, http.StatusOK)
	getJSON(t, fx.app, participant.cookie, eventsPath, http.StatusOK)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberID, map[string]any{"role": "participant"}, http.StatusOK)
	getJSON(t, fx.app, participant.cookie, eventsPath, http.StatusForbidden)

	// A participant cannot manage members or invite anyone.
	if code, _ := rawJSON(fx.app, participant.cookie, http.MethodPost, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": fx.email("friend")}); code < 400 {
		t.Fatalf("participant created an invitation: %d", code)
	}
	if code, _ := rawJSON(fx.app, participant.cookie, http.MethodPatch, "/api/workspaces/"+fx.workspaceID+"/members/"+memberID, map[string]any{"role": "crew"}); code < 400 {
		t.Fatalf("participant promoted themselves: %d", code)
	}
}
