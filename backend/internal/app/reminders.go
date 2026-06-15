package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type reminderEventDTO struct {
	ID                  string  `json:"id"`
	WorkspaceID         string  `json:"workspaceId"`
	EventID             *string `json:"eventId,omitempty"`
	SourceType          string  `json:"sourceType"`
	SourceID            string  `json:"sourceId"`
	ReminderType        string  `json:"reminderType"`
	RecipientEmail      string  `json:"recipientEmail"`
	DueAt               string  `json:"dueAt"`
	NotificationEventID *string `json:"notificationEventId,omitempty"`
	Status              string  `json:"status"`
	Subject             string  `json:"subject"`
	Preview             string  `json:"preview"`
	CreatedAt           string  `json:"createdAt"`
}

type reminderEventRow struct {
	ID                  string
	WorkspaceID         string
	EventID             sql.NullString
	SourceType          string
	SourceID            string
	ReminderType        string
	RecipientEmail      string
	DueAt               time.Time
	NotificationEventID sql.NullString
	Status              string
	Subject             string
	Preview             string
	CreatedAt           time.Time
}

func (a *App) handleListWorkspaceReminders(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	if _, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner", "member"); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	items, err := a.listReminderEvents(r.Context(), `
		select id, workspace_id, event_id, source_type, source_id, reminder_type,
		       recipient_email, due_at, notification_event_id, status, subject, preview, created_at
		from reminder_events
		where workspace_id = $1
		order by created_at desc, id desc
		limit 100
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load reminders")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleListEventReminders(w http.ResponseWriter, r *http.Request) {
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
	items, err := a.listReminderEvents(r.Context(), `
		select id, workspace_id, event_id, source_type, source_id, reminder_type,
		       recipient_email, due_at, notification_event_id, status, subject, preview, created_at
		from reminder_events
		where event_id = $1
		order by created_at desc, id desc
		limit 100
	`, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load reminders")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) listReminderEvents(ctx context.Context, query string, args ...any) ([]reminderEventDTO, error) {
	rows, err := a.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]reminderEventDTO, 0, 100)
	for rows.Next() {
		var row reminderEventRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.EventID, &row.SourceType, &row.SourceID, &row.ReminderType, &row.RecipientEmail, &row.DueAt, &row.NotificationEventID, &row.Status, &row.Subject, &row.Preview, &row.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, reminderEventDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func reminderEventDTOFromRow(row reminderEventRow) reminderEventDTO {
	dto := reminderEventDTO{
		ID:             row.ID,
		WorkspaceID:    row.WorkspaceID,
		SourceType:     row.SourceType,
		SourceID:       row.SourceID,
		ReminderType:   row.ReminderType,
		RecipientEmail: row.RecipientEmail,
		DueAt:          row.DueAt.UTC().Format(time.RFC3339Nano),
		Status:         row.Status,
		Subject:        row.Subject,
		Preview:        row.Preview,
		CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if row.EventID.Valid {
		eventID := row.EventID.String
		dto.EventID = &eventID
	}
	if row.NotificationEventID.Valid {
		notificationEventID := row.NotificationEventID.String
		dto.NotificationEventID = &notificationEventID
	}
	return dto
}
