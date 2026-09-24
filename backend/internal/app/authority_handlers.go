package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- Member role, expiry and revocation -------------------------------

type updateWorkspaceMemberRequest struct {
	Role      *string `json:"role"`
	ExpiresAt *string `json:"expiresAt"`
}

type memberAuthorityDTO struct {
	ID        string  `json:"id"`
	Role      string  `json:"role"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	RevokedAt *string `json:"revokedAt,omitempty"`
}

func formatNullableTime(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC().Format(time.RFC3339Nano)
	return &v
}

// handleUpdateWorkspaceMember changes a member's role and/or expiry. It is
// restricted to callers holding permManageMembers (owner only, per the
// permission matrix in docs/development/authority-model.md), and it
// enforces the last-owner guard: the sole active owner cannot be demoted.
func (a *App) handleUpdateWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, ok := a.requirePermission(r, workspaceID, permManageMembers)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	memberID := r.PathValue("memberID")
	if memberID == "" {
		writeError(w, http.StatusBadRequest, "member id is required")
		return
	}

	var req updateWorkspaceMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Role != nil && !isAssignableRole(*req.Role) {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}
	var expiresAt sql.NullTime
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			expiresAt = sql.NullTime{}
		} else {
			parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				writeError(w, http.StatusBadRequest, "expiresAt must be RFC3339")
				return
			}
			expiresAt = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if err := tx.QueryRow(r.Context(), `
		select id from workspaces where id = $1 for update
	`, workspaceID).Scan(new(string)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not lock workspace")
		return
	}

	var current memberAuthorityDTO
	var currentRemoved sql.NullTime
	err = tx.QueryRow(r.Context(), `
		select id, role, removed_at
		from workspace_members
		where workspace_id = $1 and id = $2
		for update
	`, workspaceID, memberID).Scan(&current.ID, &current.Role, &currentRemoved)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load member")
		return
	}
	if currentRemoved.Valid {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}

	newRole := current.Role
	if req.Role != nil {
		newRole = *req.Role
	}
	if current.Role == roleOwner && newRole != roleOwner {
		owners, err := activeOwnerCountTx(r.Context(), tx, workspaceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not count owners")
			return
		}
		if owners <= 1 {
			writeError(w, http.StatusConflict, "cannot demote the last owner")
			return
		}
	}

	var updated memberAuthorityDTO
	var updatedExpiresAt, updatedRevokedAt sql.NullTime
	if err := tx.QueryRow(r.Context(), `
		update workspace_members
		set role = $1,
		    expires_at = case when $2 then $3 else expires_at end
		where id = $4
		returning id, role, expires_at, revoked_at
	`, newRole, req.ExpiresAt != nil, expiresAt, memberID).Scan(
		&updated.ID, &updated.Role, &updatedExpiresAt, &updatedRevokedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update member")
		return
	}
	updated.ExpiresAt = formatNullableTime(updatedExpiresAt)
	updated.RevokedAt = formatNullableTime(updatedRevokedAt)

	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "workspace.member.role_changed", "workspace_member", memberID, map[string]any{
		"workspaceId": workspaceID,
		"fromRole":    current.Role,
		"toRole":      newRole,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save member")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// handleRevokeWorkspaceMember revokes a member's access without removing
// their membership row, preserving the historical record while treating
// them as absent for every authority check in this package. Restricted to
// permManageMembers; enforces the last-owner guard.
func (a *App) handleRevokeWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, ok := a.requirePermission(r, workspaceID, permManageMembers)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	memberID := r.PathValue("memberID")
	if memberID == "" {
		writeError(w, http.StatusBadRequest, "member id is required")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if err := tx.QueryRow(r.Context(), `
		select id from workspaces where id = $1 for update
	`, workspaceID).Scan(new(string)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not lock workspace")
		return
	}

	var targetRole string
	var targetRemoved, targetRevoked sql.NullTime
	err = tx.QueryRow(r.Context(), `
		select role, removed_at, revoked_at
		from workspace_members
		where workspace_id = $1 and id = $2
		for update
	`, workspaceID, memberID).Scan(&targetRole, &targetRemoved, &targetRevoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load member")
		return
	}
	if targetRemoved.Valid {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	if targetRevoked.Valid {
		writeError(w, http.StatusConflict, "member already revoked")
		return
	}

	if targetRole == roleOwner {
		owners, err := activeOwnerCountTx(r.Context(), tx, workspaceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not count owners")
			return
		}
		if owners <= 1 {
			writeError(w, http.StatusConflict, "cannot revoke the last owner")
			return
		}
	}

	if _, err := tx.Exec(r.Context(), `
		update workspace_members
		set revoked_at = now(), revoked_by_person_id = $1
		where id = $2
	`, actorID, memberID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke member")
		return
	}

	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "workspace.member.revoked", "workspace_member", memberID, map[string]any{
		"workspaceId": workspaceID,
		"role":        targetRole,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save revocation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- Creator delegations ------------------------------------------------

type creatorDelegationDTO struct {
	ID                string   `json:"id"`
	WorkspaceID       string   `json:"workspaceId"`
	CulturalProfileID string   `json:"culturalProfileId"`
	Scope             []string `json:"scope"`
	GrantedByPersonID string   `json:"grantedByPersonId"`
	GrantedAt         string   `json:"grantedAt"`
	ExpiresAt         *string  `json:"expiresAt,omitempty"`
	RevokedAt         *string  `json:"revokedAt,omitempty"`
	RevokedByPersonID *string  `json:"revokedByPersonId,omitempty"`
}

type createDelegationRequest struct {
	CulturalProfileID string   `json:"culturalProfileId"`
	Scope             []string `json:"scope"`
	ExpiresAt         *string  `json:"expiresAt"`
}

func scanDelegation(row pgx.Row) (creatorDelegationDTO, error) {
	var dto creatorDelegationDTO
	var grantedAt time.Time
	var expiresAt, revokedAt sql.NullTime
	var revokedBy sql.NullString
	if err := row.Scan(&dto.ID, &dto.WorkspaceID, &dto.CulturalProfileID, &dto.Scope,
		&dto.GrantedByPersonID, &grantedAt, &expiresAt, &revokedAt, &revokedBy); err != nil {
		return creatorDelegationDTO{}, err
	}
	dto.GrantedAt = grantedAt.UTC().Format(time.RFC3339Nano)
	dto.ExpiresAt = formatNullableTime(expiresAt)
	dto.RevokedAt = formatNullableTime(revokedAt)
	if revokedBy.Valid {
		dto.RevokedByPersonID = &revokedBy.String
	}
	if dto.Scope == nil {
		dto.Scope = []string{}
	}
	return dto, nil
}

const delegationColumns = `
	id, workspace_id, cultural_profile_id, scope,
	granted_by_person_id, granted_at, expires_at, revoked_at, revoked_by_person_id
`

// handleListDelegations lists every creator delegation ever granted in the
// workspace, active or not, so an owner/organizer can audit the delegation
// history. Restricted to permManageDelegations.
func (a *App) handleListDelegations(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	if _, ok := a.requirePermission(r, workspaceID, permManageDelegations); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		select `+delegationColumns+`
		from creator_delegations
		where workspace_id = $1
		order by granted_at desc
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load delegations")
		return
	}
	defer rows.Close()

	items := make([]creatorDelegationDTO, 0)
	for rows.Next() {
		dto, err := scanDelegation(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load delegations")
			return
		}
		items = append(items, dto)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load delegations")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleCreateDelegation grants a workspace the scoped, time-boxed right
// to act for a cultural profile. Restricted to permManageDelegations
// (owner, organizer).
func (a *App) handleCreateDelegation(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, ok := a.requirePermission(r, workspaceID, permManageDelegations)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createDelegationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.CulturalProfileID == "" {
		writeError(w, http.StatusBadRequest, "culturalProfileId is required")
		return
	}
	profile, err := a.loadCulturalProfile(r.Context(), req.CulturalProfileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cultural profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load cultural profile")
		return
	}
	if profile.WorkspaceID != workspaceID {
		writeError(w, http.StatusNotFound, "cultural profile not found")
		return
	}

	var expiresAt sql.NullTime
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "expiresAt must be RFC3339")
			return
		}
		expiresAt = sql.NullTime{Time: parsed, Valid: true}
	}
	scope := req.Scope
	if scope == nil {
		scope = []string{}
	}

	row := a.db.QueryRow(r.Context(), `
		insert into creator_delegations (workspace_id, cultural_profile_id, scope, granted_by_person_id, expires_at)
		values ($1, $2, $3, $4, $5)
		returning `+delegationColumns+`
	`, workspaceID, req.CulturalProfileID, scope, actorID, expiresAt)
	dto, err := scanDelegation(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create delegation")
		return
	}

	if err := a.audit(r.Context(), actorID, "creator_delegation.created", "creator_delegation", dto.ID, map[string]any{
		"workspaceId":       workspaceID,
		"culturalProfileId": req.CulturalProfileID,
		"scope":             scope,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// handleRevokeDelegation revokes a creator delegation. Restricted to
// permManageDelegations.
func (a *App) handleRevokeDelegation(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, ok := a.requirePermission(r, workspaceID, permManageDelegations)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	delegationID := r.PathValue("delegationID")
	if delegationID == "" {
		writeError(w, http.StatusBadRequest, "delegation id is required")
		return
	}

	result, err := a.db.Exec(r.Context(), `
		update creator_delegations
		set revoked_at = now(), revoked_by_person_id = $1, updated_at = now()
		where id = $2 and workspace_id = $3 and revoked_at is null
	`, actorID, delegationID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not revoke delegation")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "delegation not found")
		return
	}

	if err := a.audit(r.Context(), actorID, "creator_delegation.revoked", "creator_delegation", delegationID, map[string]any{
		"workspaceId": workspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
