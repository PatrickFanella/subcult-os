package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type eventRoleDTO struct {
	ID          string `json:"id"`
	EventID     string `json:"eventId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity"`
	Public      bool   `json:"public"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type eventRoleRow struct {
	ID                string
	EventID           string
	Name              string
	Description       string
	Capacity          int
	Public            bool
	Active            bool
	CreatedByPersonID string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type createEventRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity"`
	Public      *bool  `json:"public"`
}

func (a *App) handleListEventRoles(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member"); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	roles, err := a.loadEventRoles(r.Context(), event.ID, false, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load roles")
		return
	}
	writeJSON(w, http.StatusOK, eventRoleDTOsFromRows(roles))
}

func (a *App) handleCreateEventRole(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}
	actorID, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createEventRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Capacity < 0 {
		writeError(w, http.StatusBadRequest, "capacity must be non-negative")
		return
	}
	rolePublic := true
	if req.Public != nil {
		rolePublic = *req.Public
	}

	var role eventRoleRow
	if err := a.db.QueryRow(r.Context(), `
		insert into event_roles (event_id, name, description, capacity, "public", active, created_by_person_id)
		values ($1, $2, $3, $4, $5, true, $6)
		returning id, event_id, name, description, capacity, "public", active, created_by_person_id, created_at, updated_at
	`, event.ID, name, description, req.Capacity, rolePublic, actorID).Scan(&role.ID, &role.EventID, &role.Name, &role.Description, &role.Capacity, &role.Public, &role.Active, &role.CreatedByPersonID, &role.CreatedAt, &role.UpdatedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create role")
		return
	}

	writeJSON(w, http.StatusOK, eventRoleDTOFromRow(role))
}

func (a *App) handleListPublicEventRoles(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	event, err := a.loadPublishedEventBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}

	roles, err := a.loadEventRoles(r.Context(), event.ID, true, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load roles")
		return
	}
	writeJSON(w, http.StatusOK, eventRoleDTOsFromRows(roles))
}

func (a *App) loadEventRoles(ctx context.Context, eventID string, publicOnly, activeOnly bool) ([]eventRoleRow, error) {
	query := `
		select id, event_id, name, description, capacity, "public", active, created_by_person_id, created_at, updated_at
		from event_roles
		where event_id = $1`
	if publicOnly {
		query += ` and "public" = true`
	}
	if activeOnly {
		query += ` and active = true`
	}
	query += ` order by created_at asc, name asc, id asc`

	rows, err := a.db.Query(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]eventRoleRow, 0)
	for rows.Next() {
		var role eventRoleRow
		if err := rows.Scan(&role.ID, &role.EventID, &role.Name, &role.Description, &role.Capacity, &role.Public, &role.Active, &role.CreatedByPersonID, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func eventRoleDTOFromRow(row eventRoleRow) eventRoleDTO {
	return eventRoleDTO{
		ID:          row.ID,
		EventID:     row.EventID,
		Name:        row.Name,
		Description: row.Description,
		Capacity:    row.Capacity,
		Public:      row.Public,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func eventRoleDTOsFromRows(rows []eventRoleRow) []eventRoleDTO {
	roles := make([]eventRoleDTO, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, eventRoleDTOFromRow(row))
	}
	return roles
}
