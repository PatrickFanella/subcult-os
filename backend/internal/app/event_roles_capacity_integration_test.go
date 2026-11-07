package app

import (
	"net/http"
	"testing"
)

// TestRoleCapacityZeroMeansNoLimit follows the role setup contract: a
// capacity of zero is no limit. Accepting applications on such a role must
// succeed, and changing it to a real limit must still respect the accepted
// count.
func TestRoleCapacityZeroMeansNoLimit(t *testing.T) {
	fx := newLifecycleFixture(t)
	eventID := mustString(t, createEvent(t, fx, "Open call", 10), "id")
	slug := mustString(t, publishEvent(t, fx, eventID), "publicSlug")

	role := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/roles", map[string]any{"name": "Crew", "description": "Any help welcome.", "capacity": 0, "public": true}, http.StatusOK)
	roleID := mustString(t, role.JSON, "id")
	rolePath := "/api/events/" + eventID + "/roles/" + roleID

	var applicationIDs []string
	for _, applicant := range []string{"alex", "blair", "casey"} {
		application := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/role-applications", map[string]any{"roleId": roleID, "applicantName": applicant, "applicantEmail": applicant + "@example.com"}, http.StatusOK)
		applicationIDs = append(applicationIDs, mustString(t, application.JSON, "id"))
	}
	for _, id := range applicationIDs {
		patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/role-applications/"+id, map[string]any{"status": "accepted"}, http.StatusOK)
	}

	// No limit to a limit below the accepted count is refused; at or above
	// it is allowed, and the limit then applies.
	patchJSON(t, fx.app, fx.ownerCookie, rolePath, map[string]any{"capacity": 2}, http.StatusConflict)
	patchJSON(t, fx.app, fx.ownerCookie, rolePath, map[string]any{"capacity": 3}, http.StatusOK)
	extra := postJSON(t, fx.app, nil, "/api/public/events/"+slug+"/role-applications", map[string]any{"roleId": roleID, "applicantName": "dana", "applicantEmail": "dana@example.com"}, http.StatusOK)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/role-applications/"+mustString(t, extra.JSON, "id"), map[string]any{"status": "accepted"}, http.StatusConflict)

	// Back to no limit with accepted applications in place.
	patchJSON(t, fx.app, fx.ownerCookie, rolePath, map[string]any{"capacity": 0}, http.StatusOK)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/role-applications/"+mustString(t, extra.JSON, "id"), map[string]any{"status": "accepted"}, http.StatusOK)
}
