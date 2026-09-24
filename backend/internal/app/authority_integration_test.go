package app

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

// --- Migration shape --------------------------------------------------

func TestAuthorityMigrationWidensRoleAndAddsColumns(t *testing.T) {
	ctx := t.Context()
	db := newMigrationTestPool(t)
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	version, err := CurrentSchemaVersion(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if version != 10 {
		t.Fatalf("schema version = %d, want 10", version)
	}

	for _, table := range []string{"creator_delegations"} {
		var exists bool
		if err := db.QueryRow(ctx, `
			select exists (
				select 1 from information_schema.tables
				where table_schema = current_schema() and table_name = $1
			)
		`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("expected table %q to exist after RunMigrations", table)
		}
	}

	for _, column := range []string{"expires_at", "revoked_at", "revoked_by_person_id"} {
		var exists bool
		if err := db.QueryRow(ctx, `
			select exists (
				select 1 from information_schema.columns
				where table_schema = current_schema()
				  and table_name = 'workspace_members'
				  and column_name = $1
			)
		`, column).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("expected workspace_members.%s to exist after RunMigrations", column)
		}
	}

	// The widened role check still accepts the two legacy values used by
	// every pre-AUTH-01 row and call site.
	for _, role := range []string{"owner", "organizer", "finance", "door", "crew", "member"} {
		if _, err := db.Exec(ctx, `
			insert into workspaces (id, name) values (gen_random_uuid(), 'role check fixture')
		`); err != nil {
			t.Fatal(err)
		}
		var workspaceID string
		if err := db.QueryRow(ctx, `select id from workspaces order by created_at desc limit 1`).Scan(&workspaceID); err != nil {
			t.Fatal(err)
		}
		var personID string
		if err := db.QueryRow(ctx, `
			insert into people (email, password_hash) values ($1, 'x') returning id
		`, role+"-role-check@example.test").Scan(&personID); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `
			insert into workspace_members (workspace_id, person_id, role) values ($1, $2, $3)
		`, workspaceID, personID, role); err != nil {
			t.Fatalf("role %q rejected by widened check: %v", role, err)
		}
	}
}

// --- Proof: a revoked actor cannot read private roles or authorize -----
// --- queued public writes ----------------------------------------------

func TestAuthorityRevokedMemberDeniedEverywhere(t *testing.T) {
	fx := newLifecycleFixture(t)
	event := createEvent(t, fx, "Revoked Actor Show", 100)
	eventID := mustString(t, event, "id")

	// Sanity: the member can read before revocation.
	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID, http.StatusOK)
	getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/roles", http.StatusOK)

	_, memberRowID := memberIdentity(t, fx)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberRowID+"/revoke", map[string]any{}, http.StatusOK)

	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID, http.StatusForbidden)
	getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/roles", http.StatusForbidden)
	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/contacts", http.StatusForbidden)

	memberPersonID, _ := memberIdentity(t, fx)
	if err := fx.app.authorize(t.Context(), memberPersonID, fx.workspaceID, permOperate); !errors.Is(err, ErrMembershipDenied) {
		t.Fatalf("authorize for revoked member = %v, want ErrMembershipDenied", err)
	}

	profileID := createProfile(t, fx, fx.ownerCookie)
	grantDelegation(t, fx, profileID, nil)
	if err := fx.app.authorizePublicWrite(t.Context(), memberPersonID, fx.workspaceID, profileID); !errors.Is(err, ErrMembershipDenied) {
		t.Fatalf("authorizePublicWrite for revoked member = %v, want ErrMembershipDenied", err)
	}

	// Double revoke is a conflict, not a silent success.
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberRowID+"/revoke", map[string]any{}, http.StatusConflict)
}

func TestAuthorityExpiredMembershipDeniedEverywhere(t *testing.T) {
	fx := newLifecycleFixture(t)
	memberPersonID, memberRowID := memberIdentity(t, fx)

	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberRowID, map[string]any{"expiresAt": past}, http.StatusOK)

	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID, http.StatusForbidden)

	if err := fx.app.authorize(t.Context(), memberPersonID, fx.workspaceID, permOperate); !errors.Is(err, ErrMembershipDenied) {
		t.Fatalf("authorize for expired member = %v, want ErrMembershipDenied", err)
	}
}

func TestAuthorityLastOwnerCannotBeRemovedDemotedOrRevoked(t *testing.T) {
	fx := newLifecycleFixture(t)
	ownerRowID := ownerMemberRowID(t, fx)

	patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+ownerRowID, map[string]any{"role": roleCrew}, http.StatusConflict)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+ownerRowID+"/revoke", map[string]any{}, http.StatusConflict)
	doJSON(t, http.MethodDelete, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+ownerRowID, nil, http.StatusConflict)

	// The owner role and membership are unaffected by the rejected attempts.
	getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID, http.StatusOK)
}

// --- authorizePublicWrite: PUB-AUTH's future hook -----------------------

func TestAuthorizePublicWriteDeniesEachReason(t *testing.T) {
	fx := newLifecycleFixture(t)
	profileID := createProfile(t, fx, fx.ownerCookie)
	ownerPersonID := ownerPersonID(t, fx)
	memberPersonID, _ := memberIdentity(t, fx)

	// Owner has permPublish but no delegation yet exists for this profile:
	// denied for the delegation reason specifically, not a permission
	// reason.
	if err := fx.app.authorizePublicWrite(t.Context(), ownerPersonID, fx.workspaceID, profileID); !errors.Is(err, ErrDelegationDenied) {
		t.Fatalf("missing delegation: got %v, want ErrDelegationDenied", err)
	}

	// crew/member role lacks permPublish even with a delegation present.
	grantDelegation(t, fx, profileID, nil)
	if err := fx.app.authorizePublicWrite(t.Context(), memberPersonID, fx.workspaceID, profileID); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("role without publish permission: got %v, want ErrPermissionDenied", err)
	}

	// Owner has permPublish and an active delegation: allowed.
	if err := fx.app.authorizePublicWrite(t.Context(), ownerPersonID, fx.workspaceID, profileID); err != nil {
		t.Fatalf("owner with active delegation should be authorized: %v", err)
	}

	// An expired delegation denies even an otherwise-authorized actor.
	past := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	otherProfileID := createProfile(t, fx, fx.ownerCookie)
	grantDelegation(t, fx, otherProfileID, &past)
	if err := fx.app.authorizePublicWrite(t.Context(), ownerPersonID, fx.workspaceID, otherProfileID); !errors.Is(err, ErrDelegationDenied) {
		t.Fatalf("expired delegation: got %v, want ErrDelegationDenied", err)
	}

	// A revoked delegation denies too.
	thirdProfileID := createProfile(t, fx, fx.ownerCookie)
	delegationID := grantDelegation(t, fx, thirdProfileID, nil)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/delegations/"+delegationID+"/revoke", map[string]any{}, http.StatusOK)
	if err := fx.app.authorizePublicWrite(t.Context(), ownerPersonID, fx.workspaceID, thirdProfileID); !errors.Is(err, ErrDelegationDenied) {
		t.Fatalf("revoked delegation: got %v, want ErrDelegationDenied", err)
	}

	// A revoked membership denies regardless of delegation state.
	memberRowID := memberRowIDOnly(t, fx)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberRowID+"/revoke", map[string]any{}, http.StatusOK)
	if err := fx.app.authorizePublicWrite(t.Context(), memberPersonID, fx.workspaceID, profileID); !errors.Is(err, ErrMembershipDenied) {
		t.Fatalf("revoked membership: got %v, want ErrMembershipDenied", err)
	}
}

// --- Role change and delegation endpoints, restricted to owner ---------

func TestUpdateMemberRoleRestrictedToOwner(t *testing.T) {
	fx := newLifecycleFixture(t)
	_, memberRowID := memberIdentity(t, fx)

	patchJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberRowID, map[string]any{"role": roleOrganizer}, http.StatusForbidden)
	patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberRowID, map[string]any{"role": roleOrganizer}, http.StatusOK)

	memberPersonID, _ := memberIdentity(t, fx)
	if err := fx.app.authorize(t.Context(), memberPersonID, fx.workspaceID, permManageDelegations); err != nil {
		t.Fatalf("organizer should carry permManageDelegations after role change: %v", err)
	}
}

func TestDelegationEndpointsRestrictedAndScoped(t *testing.T) {
	fx := newLifecycleFixture(t)
	profileID := createProfile(t, fx, fx.ownerCookie)

	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/delegations", map[string]any{"culturalProfileId": profileID}, http.StatusForbidden)
	getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/delegations", http.StatusForbidden)

	created := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/delegations", map[string]any{
		"culturalProfileId": profileID,
		"scope":             []string{"publish"},
	}, http.StatusOK)
	delegationID := mustString(t, created.JSON, "id")

	listed := getJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/delegations", http.StatusOK)
	items, ok := listed.JSON.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected exactly one delegation, got %#v", listed.JSON)
	}

	postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/delegations/"+delegationID+"/revoke", map[string]any{}, http.StatusForbidden)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/delegations/"+delegationID+"/revoke", map[string]any{}, http.StatusOK)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/delegations/"+delegationID+"/revoke", map[string]any{}, http.StatusNotFound)

	// A delegation cannot be created against a profile from another workspace.
	other := newLifecycleFixture(t, fx.app)
	otherProfileID := createProfile(t, other, other.ownerCookie)
	postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/delegations", map[string]any{"culturalProfileId": otherProfileID}, http.StatusNotFound)
}

// --- test helpers --------------------------------------------------------

func memberIdentity(t *testing.T, fx lifecycleFixture) (personID, memberRowID string) {
	t.Helper()
	if err := fx.app.db.QueryRow(t.Context(), `
		select wm.person_id, wm.id
		from workspace_members wm
		join people p on p.id = wm.person_id
		where wm.workspace_id = $1 and p.email = $2
	`, fx.workspaceID, fx.email("member")).Scan(&personID, &memberRowID); err != nil {
		t.Fatal(err)
	}
	return personID, memberRowID
}

func memberRowIDOnly(t *testing.T, fx lifecycleFixture) string {
	t.Helper()
	_, id := memberIdentity(t, fx)
	return id
}

func ownerMemberRowID(t *testing.T, fx lifecycleFixture) string {
	t.Helper()
	var id string
	if err := fx.app.db.QueryRow(t.Context(), `
		select id from workspace_members where workspace_id = $1 and role = 'owner' limit 1
	`, fx.workspaceID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func createProfile(t *testing.T, fx lifecycleFixture, cookie *http.Cookie) string {
	t.Helper()
	resp := postJSON(t, fx.app, cookie, "/api/workspaces/"+fx.workspaceID+"/profiles", map[string]any{
		"kind":        "creator",
		"displayName": "Fixture Creator",
	}, http.StatusOK)
	return mustString(t, resp.JSON, "id")
}

// grantDelegation creates a creator delegation for profileID, optionally
// with an explicit expiresAt (RFC3339, may already be in the past for
// expiry fixtures), and returns the delegation ID.
func grantDelegation(t *testing.T, fx lifecycleFixture, profileID string, expiresAt *string) string {
	t.Helper()
	payload := map[string]any{"culturalProfileId": profileID, "scope": []string{"publish"}}
	if expiresAt != nil {
		payload["expiresAt"] = *expiresAt
	}
	resp := postJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/delegations", payload, http.StatusOK)
	return mustString(t, resp.JSON, "id")
}

// TestPromotedRolesKeepBaselineAccessOnLegacyRoutes proves that a member
// promoted to one of the new roles still passes every legacy handler that
// spells "any member" as the literal "member" role, while owner-only
// handlers keep denying them.
func TestPromotedRolesKeepBaselineAccessOnLegacyRoutes(t *testing.T) {
	fx := newLifecycleFixture(t)
	_, memberRowID := memberIdentity(t, fx)
	event := createEvent(t, fx, "Promoted Crew Show", 100)
	eventID := mustString(t, event, "id")

	for _, role := range []string{roleOrganizer, roleFinance, roleDoor, roleCrew} {
		patchJSON(t, fx.app, fx.ownerCookie, "/api/workspaces/"+fx.workspaceID+"/members/"+memberRowID, map[string]any{"role": role}, http.StatusOK)
		getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/events", http.StatusOK)
		getJSON(t, fx.app, fx.memberCookie, "/api/events/"+eventID+"/roles", http.StatusOK)
		getJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/contacts", http.StatusOK)
		// Owner-only surface stays closed to every non-owner role.
		postJSON(t, fx.app, fx.memberCookie, "/api/workspaces/"+fx.workspaceID+"/invitations", map[string]any{"email": fx.email("nobody-" + role)}, http.StatusForbidden)
	}
}
