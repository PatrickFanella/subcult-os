package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

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

type eventRoleApplicationDTO struct {
	ID             string `json:"id"`
	EventID        string `json:"eventId"`
	RoleID         string `json:"roleId"`
	ApplicantName  string `json:"applicantName"`
	ApplicantEmail string `json:"applicantEmail"`
	Message        string `json:"message"`
	Status         string `json:"status"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type eventRoleApplicationRow struct {
	ID             string
	EventID        string
	RoleID         string
	ApplicantName  string
	ApplicantEmail string
	Message        string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type submitPublicRoleApplicationRequest struct {
	RoleID         string `json:"roleId"`
	ApplicantName  string `json:"applicantName"`
	ApplicantEmail string `json:"applicantEmail"`
	Message        string `json:"message"`
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

func (a *App) handleSubmitPublicRoleApplication(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	var req submitPublicRoleApplicationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
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

	roleID := strings.TrimSpace(req.RoleID)
	role, err := a.loadPublicActiveEventRole(r.Context(), event.ID, roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "role not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load role")
		return
	}

	applicantName := strings.TrimSpace(req.ApplicantName)
	applicantEmail := normalizeEmail(req.ApplicantEmail)
	message := strings.TrimSpace(req.Message)
	if applicantName == "" {
		writeError(w, http.StatusBadRequest, "applicant name is required")
		return
	}
	if !basicEmail(applicantEmail) {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if utf8.RuneCountInString(message) > 2000 {
		writeError(w, http.StatusBadRequest, "message must be 2000 characters or fewer")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var application eventRoleApplicationRow
	if err := tx.QueryRow(r.Context(), `
		insert into event_role_applications (event_id, role_id, applicant_name, applicant_email, message, status)
		values ($1, $2, $3, $4, $5, 'submitted')
		returning id, event_id, role_id, applicant_name, applicant_email, message, status, created_at, updated_at
	`, event.ID, role.ID, applicantName, applicantEmail, message).Scan(&application.ID, &application.EventID, &application.RoleID, &application.ApplicantName, &application.ApplicantEmail, &application.Message, &application.Status, &application.CreatedAt, &application.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "application already submitted")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create application")
		return
	}

	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, "", "role_application.submitted", "event_role_application", application.ID, map[string]any{
		"eventId":       application.EventID,
		"roleId":        application.RoleID,
		"applicationId": application.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save application")
		return
	}

	writeJSON(w, http.StatusOK, eventRoleApplicationDTOFromRow(application))
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

func (a *App) loadPublicActiveEventRole(ctx context.Context, eventID, roleID string) (eventRoleRow, error) {
	var role eventRoleRow
	if roleID == "" {
		return role, pgx.ErrNoRows
	}
	if err := a.db.QueryRow(ctx, `
		select id, event_id, name, description, capacity, "public", active, created_by_person_id, created_at, updated_at
		from event_roles
		where event_id = $1
		  and id = $2
		  and "public" = true
		  and active = true
	`, eventID, roleID).Scan(&role.ID, &role.EventID, &role.Name, &role.Description, &role.Capacity, &role.Public, &role.Active, &role.CreatedByPersonID, &role.CreatedAt, &role.UpdatedAt); err != nil {
		return eventRoleRow{}, err
	}
	return role, nil
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

func eventRoleApplicationDTOFromRow(row eventRoleApplicationRow) eventRoleApplicationDTO {
	return eventRoleApplicationDTO{
		ID:             row.ID,
		EventID:        row.EventID,
		RoleID:         row.RoleID,
		ApplicantName:  row.ApplicantName,
		ApplicantEmail: row.ApplicantEmail,
		Message:        row.Message,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func basicEmail(email string) bool {
	if email == "" || strings.Contains(email, " ") {
		return false
	}
	local, domain, ok := strings.Cut(email, "@")
	return ok && local != "" && domain != ""
}
