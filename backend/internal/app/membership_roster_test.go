package app

import (
	"net/http"
	"testing"
)

func TestWorkspaceRosterReportsCurrentMembershipAccess(t *testing.T) {
	fx := newLifecycleFixture(t)
	_, memberID := memberIdentity(t, fx)
	path := "/api/workspaces/" + fx.workspaceID
	for _, tc := range []struct {
		name, expiry, revoked, removed, state string
		memberStatus                          int
	}{
		{"active", "null", "null", "null", "active", http.StatusOK},
		{"future-expiry", "now() + interval '1 hour'", "null", "null", "active", http.StatusOK},
		{"expired", "now()", "null", "null", "expired", http.StatusForbidden},
		{"revoked", "null", "now()", "null", "revoked", http.StatusForbidden},
		{"revoked-and-expired", "now() - interval '1 hour'", "now()", "null", "revoked", http.StatusForbidden},
		{"removed", "null", "null", "now()", "", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// SQL fragments are fixed fixture constants, never request data.
			if _, err := fx.app.db.Exec(t.Context(), "update workspace_members set expires_at = "+tc.expiry+", revoked_at = "+tc.revoked+", removed_at = "+tc.removed+" where id = $1", memberID); err != nil {
				t.Fatal(err)
			}
			for _, route := range []string{path, "/api/workspaces/current"} {
				response := getJSON(t, fx.app, fx.ownerCookie, route, http.StatusOK)
				if response.Header.Get("Cache-Control") != "no-store" {
					t.Fatal("private roster must not be cached")
				}
				rows := mustObject(t, response.JSON)["members"].([]any)
				var found map[string]any
				for _, row := range rows {
					member := mustObject(t, row)
					if member["id"] == memberID {
						found = member
					}
				}
				if tc.state == "" {
					if found != nil {
						t.Fatal("removed membership remains in roster")
					}
					continue
				}
				if found == nil || found["accessState"] != tc.state {
					t.Fatalf("access state = %v, want %s", found, tc.state)
				}
				if _, present := found["expiresAt"]; present != (tc.expiry != "null") {
					t.Fatal("expiry serialization does not match persisted state")
				}
				if _, present := found["revokedAt"]; present != (tc.revoked != "null") {
					t.Fatal("revocation serialization does not match persisted state")
				}
			}
			response := getJSON(t, fx.app, fx.memberCookie, path, tc.memberStatus)
			if response.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("denied roster response must not be cached")
			}
		})
	}
}
