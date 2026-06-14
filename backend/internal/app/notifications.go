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

type notificationEventDTO struct {
	ID               string  `json:"id"`
	EventID          *string `json:"eventId,omitempty"`
	RecipientEmail   string  `json:"recipientEmail"`
	NotificationType string  `json:"notificationType"`
	RelatedType      string  `json:"relatedType"`
	RelatedID        *string `json:"relatedId,omitempty"`
	Subject          string  `json:"subject"`
	Preview          string  `json:"preview"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"createdAt"`
}

type notificationEventRow struct {
	ID               string
	EventID          sql.NullString
	RecipientEmail   string
	NotificationType string
	RelatedType      string
	RelatedID        sql.NullString
	Subject          string
	Preview          string
	Status           string
	CreatedAt        time.Time
}

type enqueueNotificationParams struct {
	WorkspaceID       string
	EventID           string
	RecipientEmail    string
	NotificationType  string
	RelatedType       string
	RelatedID         string
	IdempotencyKey    string
	Subject           string
	Body              string
	Preview           string
	CreatedByPersonID string
}

func (a *App) enqueueNotification(ctx context.Context, tx pgx.Tx, params enqueueNotificationParams) (bool, error) {
	if tx == nil {
		return false, nil
	}
	recipient := normalizeEmail(params.RecipientEmail)
	if recipient == "" || strings.TrimSpace(params.IdempotencyKey) == "" {
		return false, nil
	}

	eventID := sql.NullString{String: params.EventID, Valid: params.EventID != ""}
	relatedID := sql.NullString{String: params.RelatedID, Valid: params.RelatedID != ""}
	actorID := sql.NullString{String: params.CreatedByPersonID, Valid: params.CreatedByPersonID != ""}
	preview := strings.TrimSpace(params.Preview)
	if len([]rune(preview)) > 240 {
		preview = string([]rune(preview)[:240])
	}

	var notificationID string
	if err := tx.QueryRow(ctx, `
		insert into notification_events (
			workspace_id, event_id, recipient_email, notification_type, related_type,
			related_id, idempotency_key, email_outbox_id, subject, preview, created_by_person_id
		)
		values ($1, $2, $3, $4, $5, $6, $7, null, $8, $9, $10)
		on conflict (idempotency_key) do nothing
		returning id
	`, params.WorkspaceID, eventID, recipient, params.NotificationType, params.RelatedType, relatedID, params.IdempotencyKey, params.Subject, preview, actorID).Scan(&notificationID); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	var outboxID string
	if err := tx.QueryRow(ctx, `
		insert into email_outbox (recipient_email, subject, body, related_type, related_id)
		values ($1, $2, $3, $4, $5)
		returning id
	`, recipient, params.Subject, params.Body, params.RelatedType, relatedID).Scan(&outboxID); err != nil {
		return false, err
	}

	if _, err := tx.Exec(ctx, `update notification_events set email_outbox_id = $1 where id = $2`, outboxID, notificationID); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) handleListEventNotifications(w http.ResponseWriter, r *http.Request) {
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
		select id, event_id, recipient_email, notification_type, related_type, related_id, subject, preview, status, created_at
		from notification_events
		where event_id = $1
		order by created_at desc
		limit 50
	`, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load notifications")
		return
	}
	defer rows.Close()

	items := make([]notificationEventDTO, 0, 50)
	for rows.Next() {
		var row notificationEventRow
		if err := rows.Scan(&row.ID, &row.EventID, &row.RecipientEmail, &row.NotificationType, &row.RelatedType, &row.RelatedID, &row.Subject, &row.Preview, &row.Status, &row.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load notifications")
			return
		}
		items = append(items, notificationEventDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load notifications")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func notificationEventDTOFromRow(row notificationEventRow) notificationEventDTO {
	dto := notificationEventDTO{
		ID:               row.ID,
		RecipientEmail:   row.RecipientEmail,
		NotificationType: row.NotificationType,
		RelatedType:      row.RelatedType,
		Subject:          row.Subject,
		Preview:          row.Preview,
		Status:           row.Status,
		CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if row.EventID.Valid {
		eventID := row.EventID.String
		dto.EventID = &eventID
	}
	if row.RelatedID.Valid {
		relatedID := row.RelatedID.String
		dto.RelatedID = &relatedID
	}
	return dto
}
