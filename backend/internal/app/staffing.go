package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type createEventStaffingRequest struct {
	Title    string  `json:"title"`
	Kind     string  `json:"kind"`
	Notes    string  `json:"notes"`
	StartsAt *string `json:"startsAt"`
	EndsAt   *string `json:"endsAt"`
}

type eventStaffingItemDTO struct {
	ID                    string  `json:"id"`
	EventID               string  `json:"eventId"`
	Title                 string  `json:"title"`
	Kind                  string  `json:"kind"`
	Notes                 string  `json:"notes"`
	StartsAt              *string `json:"startsAt,omitempty"`
	EndsAt                *string `json:"endsAt,omitempty"`
	AssignedPersonID      *string `json:"assignedPersonId,omitempty"`
	AssignedApplicationID *string `json:"assignedApplicationId,omitempty"`
	AssigneeName          *string `json:"assigneeName,omitempty"`
	Status                string  `json:"status"`
	CreatedAt             string  `json:"createdAt"`
	UpdatedAt             string  `json:"updatedAt"`
	CompletedAt           *string `json:"completedAt,omitempty"`
	CompletedByPersonID   *string `json:"completedByPersonId,omitempty"`
}

type eventStaffingItemRow struct {
	ID                    string
	EventID               string
	Title                 string
	Kind                  string
	Notes                 string
	StartsAt              sql.NullTime
	EndsAt                sql.NullTime
	AssignedPersonID      sql.NullString
	AssignedApplicationID sql.NullString
	AssigneeName          sql.NullString
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	CompletedAt           sql.NullTime
	CompletedByPersonID   sql.NullString
}

func (a *App) handleListEventStaffing(w http.ResponseWriter, r *http.Request) {
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

	rows, err := a.db.Query(r.Context(), `
		select esi.id, esi.event_id, esi.title, esi.kind, esi.notes, esi.starts_at, esi.ends_at,
		       esi.assigned_person_id, esi.assigned_application_id,
		       coalesce(nullif(trim(p.display_name), ''), p.email, era.applicant_name) as assignee_name,
		       esi.status, esi.created_at, esi.updated_at, esi.completed_at, esi.completed_by_person_id
		from event_staffing_items esi
		left join people p on p.id = esi.assigned_person_id
		left join event_role_applications era on era.id = esi.assigned_application_id
		where esi.event_id = $1
		order by case esi.status
			when 'open' then 0
			when 'assigned' then 1
			when 'completed' then 2
			when 'cancelled' then 3
			else 4
		end, esi.starts_at nulls last, esi.created_at, esi.id
	`, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load staffing")
		return
	}
	defer rows.Close()

	items := make([]eventStaffingItemDTO, 0)
	for rows.Next() {
		var row eventStaffingItemRow
		if err := rows.Scan(&row.ID, &row.EventID, &row.Title, &row.Kind, &row.Notes, &row.StartsAt, &row.EndsAt, &row.AssignedPersonID, &row.AssignedApplicationID, &row.AssigneeName, &row.Status, &row.CreatedAt, &row.UpdatedAt, &row.CompletedAt, &row.CompletedByPersonID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load staffing")
			return
		}
		items = append(items, eventStaffingItemDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load staffing")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleCreateEventStaffing(w http.ResponseWriter, r *http.Request) {
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

	var req createEventStaffingRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	title := strings.TrimSpace(req.Title)
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	notes := strings.TrimSpace(req.Notes)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	switch kind {
	case "task", "shift":
	default:
		writeError(w, http.StatusBadRequest, "kind must be task or shift")
		return
	}
	if utf8.RuneCountInString(notes) > 2000 {
		writeError(w, http.StatusBadRequest, "notes must be 2000 characters or fewer")
		return
	}
	startsAt, err := parseOptionalRFC3339Time(req.StartsAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startsAt")
		return
	}
	endsAt, err := parseOptionalRFC3339Time(req.EndsAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid endsAt")
		return
	}
	if startsAt != nil && endsAt != nil && endsAt.Before(*startsAt) {
		writeError(w, http.StatusBadRequest, "endsAt must be greater than or equal to startsAt")
		return
	}
	if event.Status == "end_of_night" {
		writeError(w, http.StatusConflict, "event is closed")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var startsAtArg any
	if startsAt != nil {
		startsAtArg = *startsAt
	}
	var endsAtArg any
	if endsAt != nil {
		endsAtArg = *endsAt
	}

	var row eventStaffingItemRow
	if err := tx.QueryRow(r.Context(), `
		insert into event_staffing_items (
			event_id, title, kind, notes, starts_at, ends_at, status, created_by_person_id
		)
		values ($1, $2, $3, $4, $5, $6, 'open', $7)
		returning id, event_id, title, kind, notes, starts_at, ends_at,
		          assigned_person_id, assigned_application_id,
		          null as assignee_name,
		          status, created_at, updated_at, completed_at, completed_by_person_id
	`, event.ID, title, kind, notes, startsAtArg, endsAtArg, actorID).Scan(
		&row.ID, &row.EventID, &row.Title, &row.Kind, &row.Notes, &row.StartsAt, &row.EndsAt,
		&row.AssignedPersonID, &row.AssignedApplicationID, &row.AssigneeName, &row.Status, &row.CreatedAt, &row.UpdatedAt, &row.CompletedAt, &row.CompletedByPersonID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create staffing item")
		return
	}

	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "staffing.created", "event_staffing_item", row.ID, map[string]any{
		"eventId":        row.EventID,
		"staffingItemId": row.ID,
		"kind":           row.Kind,
		"status":         row.Status,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save staffing item")
		return
	}

	writeJSON(w, http.StatusOK, eventStaffingItemDTOFromRow(row))
}

func eventStaffingItemDTOFromRow(row eventStaffingItemRow) eventStaffingItemDTO {
	dto := eventStaffingItemDTO{
		ID:        row.ID,
		EventID:   row.EventID,
		Title:     row.Title,
		Kind:      row.Kind,
		Notes:     row.Notes,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	dto.StartsAt = nullableTimeString(row.StartsAt)
	dto.EndsAt = nullableTimeString(row.EndsAt)
	dto.AssignedPersonID = nullableString(row.AssignedPersonID)
	dto.AssignedApplicationID = nullableString(row.AssignedApplicationID)
	dto.AssigneeName = nullableString(row.AssigneeName)
	dto.CompletedAt = nullableTimeString(row.CompletedAt)
	dto.CompletedByPersonID = nullableString(row.CompletedByPersonID)
	return dto
}

func parseOptionalRFC3339Time(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := parseRFC3339Time(*value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
