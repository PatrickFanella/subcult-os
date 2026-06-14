package app

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

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
