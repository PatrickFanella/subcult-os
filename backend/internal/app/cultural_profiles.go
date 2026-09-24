package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Profile kinds. "act" is a profile kind, not a separate table, per D6's
// recommended starting point in docs/development/decisions.md.
const (
	profileKindCreator    = "creator"
	profileKindCollective = "collective"
	profileKindAct        = "act"
)

func isValidProfileKind(kind string) bool {
	switch kind {
	case profileKindCreator, profileKindCollective, profileKindAct:
		return true
	default:
		return false
	}
}

type culturalProfileDTO struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspaceId"`
	Kind              string  `json:"kind"`
	DisplayName       string  `json:"displayName"`
	Description       string  `json:"description"`
	PublicURI         *string `json:"publicUri,omitempty"`
	PublicCID         *string `json:"publicCid,omitempty"`
	CreatedByPersonID string  `json:"createdByPersonId"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type culturalProfileRow struct {
	ID                string
	WorkspaceID       string
	Kind              string
	DisplayName       string
	Description       string
	PublicURI         sql.NullString
	PublicCID         sql.NullString
	CreatedByPersonID string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func culturalProfileDTOFromRow(row culturalProfileRow) culturalProfileDTO {
	return culturalProfileDTO{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Kind:              row.Kind,
		DisplayName:       row.DisplayName,
		Description:       row.Description,
		PublicURI:         nullableString(row.PublicURI),
		PublicCID:         nullableString(row.PublicCID),
		CreatedByPersonID: row.CreatedByPersonID,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type createCulturalProfileRequest struct {
	Kind        string `json:"kind"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type updateCulturalProfileRequest struct {
	Kind        *string `json:"kind"`
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
}

func (a *App) loadCulturalProfile(ctx context.Context, profileID string) (culturalProfileRow, error) {
	var row culturalProfileRow
	if profileID == "" {
		return row, pgx.ErrNoRows
	}
	err := a.db.QueryRow(ctx, `
		select id, workspace_id, kind, display_name, description, public_uri, public_cid,
		       created_by_person_id, created_at, updated_at
		from cultural_profiles
		where id = $1
	`, profileID).Scan(&row.ID, &row.WorkspaceID, &row.Kind, &row.DisplayName, &row.Description,
		&row.PublicURI, &row.PublicCID, &row.CreatedByPersonID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return culturalProfileRow{}, err
	}
	return row, nil
}

func (a *App) handleListCulturalProfiles(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	if _, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner", "member"); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		select id, workspace_id, kind, display_name, description, public_uri, public_cid,
		       created_by_person_id, created_at, updated_at
		from cultural_profiles
		where workspace_id = $1
		order by created_at desc, id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load profiles")
		return
	}
	defer rows.Close()

	items := make([]culturalProfileDTO, 0)
	for rows.Next() {
		var row culturalProfileRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.Kind, &row.DisplayName, &row.Description,
			&row.PublicURI, &row.PublicCID, &row.CreatedByPersonID, &row.CreatedAt, &row.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load profiles")
			return
		}
		items = append(items, culturalProfileDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load profiles")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleCreateCulturalProfile(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createCulturalProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = profileKindCreator
	}
	if !isValidProfileKind(kind) {
		writeError(w, http.StatusBadRequest, "invalid kind")
		return
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" || len(displayName) > 500 {
		writeError(w, http.StatusBadRequest, "displayName is required and must be 1-500 characters")
		return
	}
	description := strings.TrimSpace(req.Description)
	if len(description) > 3000 {
		writeError(w, http.StatusBadRequest, "description must be 3000 characters or fewer")
		return
	}

	var row culturalProfileRow
	err := a.db.QueryRow(r.Context(), `
		insert into cultural_profiles (workspace_id, kind, display_name, description, created_by_person_id)
		values ($1, $2, $3, $4, $5)
		returning id, workspace_id, kind, display_name, description, public_uri, public_cid,
		          created_by_person_id, created_at, updated_at
	`, workspaceID, kind, displayName, description, actorID).Scan(
		&row.ID, &row.WorkspaceID, &row.Kind, &row.DisplayName, &row.Description,
		&row.PublicURI, &row.PublicCID, &row.CreatedByPersonID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create profile")
		return
	}
	if err := a.audit(r.Context(), actorID, "cultural_profile.created", "cultural_profile", row.ID, map[string]any{
		"workspaceId": workspaceID,
		"kind":        kind,
		"displayName": displayName,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	writeJSON(w, http.StatusOK, culturalProfileDTOFromRow(row))
}

func (a *App) handleUpdateCulturalProfile(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	profile, err := a.loadCulturalProfile(r.Context(), r.PathValue("profileID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load profile")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, profile.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req updateCulturalProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	kind := profile.Kind
	if req.Kind != nil {
		kind = strings.TrimSpace(*req.Kind)
		if !isValidProfileKind(kind) {
			writeError(w, http.StatusBadRequest, "invalid kind")
			return
		}
	}
	displayName := profile.DisplayName
	if req.DisplayName != nil {
		displayName = strings.TrimSpace(*req.DisplayName)
		if displayName == "" || len(displayName) > 500 {
			writeError(w, http.StatusBadRequest, "displayName must be 1-500 characters")
			return
		}
	}
	description := profile.Description
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
		if len(description) > 3000 {
			writeError(w, http.StatusBadRequest, "description must be 3000 characters or fewer")
			return
		}
	}

	var row culturalProfileRow
	err = a.db.QueryRow(r.Context(), `
		update cultural_profiles
		set kind = $2, display_name = $3, description = $4, updated_at = now()
		where id = $1
		returning id, workspace_id, kind, display_name, description, public_uri, public_cid,
		          created_by_person_id, created_at, updated_at
	`, profile.ID, kind, displayName, description).Scan(
		&row.ID, &row.WorkspaceID, &row.Kind, &row.DisplayName, &row.Description,
		&row.PublicURI, &row.PublicCID, &row.CreatedByPersonID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	if err := a.audit(r.Context(), actorID, "cultural_profile.updated", "cultural_profile", row.ID, map[string]any{
		"workspaceId": row.WorkspaceID,
		"kind":        kind,
		"displayName": displayName,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	writeJSON(w, http.StatusOK, culturalProfileDTOFromRow(row))
}
