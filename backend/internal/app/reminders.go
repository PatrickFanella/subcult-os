package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

type reminderSweepRequest struct {
	Now *string `json:"now,omitempty"`
}

type reminderSweepResult struct {
	CreatedCount int `json:"createdCount"`
	SkippedCount int `json:"skippedCount"`
}

type reminderActionParams struct {
	WorkspaceID       string
	EventID           string
	SourceType        string
	SourceID          string
	ReminderType      string
	RecipientEmail    string
	DueAt             time.Time
	IdempotencyKey    string
	NotificationType  string
	RelatedType       string
	RelatedID         string
	Subject           string
	Body              string
	Preview           string
	CreatedByPersonID string
}

type commitmentReminderRow struct {
	ID            string
	WorkspaceID   string
	EventID       sql.NullString
	Title         string
	DueAt         time.Time
	OwnerPersonID sql.NullString
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

func (a *App) handleSweepWorkspaceReminders(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req reminderSweepRequest
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now().UTC()
	if req.Now != nil {
		parsed, err := parseRFC3339Time(*req.Now)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid now")
			return
		}
		now = parsed.UTC()
	}

	result, err := a.runWorkspaceReminderSweep(r.Context(), workspaceID, actorID, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not sweep reminders")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) runWorkspaceReminderSweep(ctx context.Context, workspaceID, actorID string, now time.Time) (reminderSweepResult, error) {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return reminderSweepResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		select id, workspace_id, event_id, title, due_at, owner_person_id
		from commitments
		where workspace_id = $1
		  and status = 'open'
		  and due_at is not null
		  and due_at <= $2
		order by due_at asc, created_at asc, id asc
	`, workspaceID, now.UTC())
	if err != nil {
		return reminderSweepResult{}, err
	}
	defer rows.Close()

	result := reminderSweepResult{}
	for rows.Next() {
		var row commitmentReminderRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.EventID, &row.Title, &row.DueAt, &row.OwnerPersonID); err != nil {
			return reminderSweepResult{}, err
		}

		recipients, err := a.commitmentReminderRecipients(ctx, tx, workspaceID, row.OwnerPersonID)
		if err != nil {
			return reminderSweepResult{}, err
		}
		if len(recipients) == 0 {
			result.SkippedCount++
			continue
		}

		subject, preview, body := commitmentReminderCopy(row.Title, row.DueAt)
		eventID := ""
		if row.EventID.Valid {
			eventID = row.EventID.String
		}

		for _, recipient := range recipients {
			created, err := a.createReminderAndNotification(ctx, tx, reminderActionParams{
				WorkspaceID:       workspaceID,
				EventID:           eventID,
				SourceType:        "commitment",
				SourceID:          row.ID,
				ReminderType:      "commitment.due",
				RecipientEmail:    recipient,
				DueAt:             row.DueAt.UTC(),
				IdempotencyKey:    "commitment_due:" + row.ID + ":" + recipient,
				NotificationType:  "commitment.due",
				RelatedType:       "commitment",
				RelatedID:         row.ID,
				Subject:           subject,
				Body:              body,
				Preview:           preview,
				CreatedByPersonID: actorID,
			})
			if err != nil {
				return reminderSweepResult{}, err
			}
			if created {
				result.CreatedCount++
			} else {
				result.SkippedCount++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return reminderSweepResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return reminderSweepResult{}, err
	}
	return result, nil
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

func (a *App) createReminderAndNotification(ctx context.Context, tx pgx.Tx, params reminderActionParams) (bool, error) {
	if tx == nil {
		return false, nil
	}
	recipient := normalizeEmail(params.RecipientEmail)
	if recipient == "" || strings.TrimSpace(params.IdempotencyKey) == "" {
		return false, nil
	}

	eventID := sql.NullString{String: params.EventID, Valid: params.EventID != ""}
	createdBy := sql.NullString{String: params.CreatedByPersonID, Valid: params.CreatedByPersonID != ""}
	dueAt := params.DueAt.UTC()

	var reminderID string
	if err := tx.QueryRow(ctx, `
		insert into reminder_events (
			workspace_id, event_id, source_type, source_id, reminder_type,
			recipient_email, due_at, idempotency_key, status, subject, preview, created_by_person_id
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, 'queued', $9, $10, $11)
		on conflict (idempotency_key) do nothing
		returning id
	`, params.WorkspaceID, eventID, params.SourceType, params.SourceID, params.ReminderType, recipient, dueAt, params.IdempotencyKey, params.Subject, params.Preview, createdBy).Scan(&reminderID); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	created, err := a.enqueueNotification(ctx, tx, enqueueNotificationParams{
		WorkspaceID:       params.WorkspaceID,
		EventID:           params.EventID,
		RecipientEmail:    recipient,
		NotificationType:  params.NotificationType,
		RelatedType:       params.RelatedType,
		RelatedID:         params.RelatedID,
		IdempotencyKey:    params.IdempotencyKey,
		Subject:           params.Subject,
		Body:              params.Body,
		Preview:           params.Preview,
		CreatedByPersonID: params.CreatedByPersonID,
	})
	if err != nil {
		return false, err
	}
	if !created {
		if _, err := tx.Exec(ctx, `delete from reminder_events where id = $1`, reminderID); err != nil {
			return false, err
		}
		return false, nil
	}

	var notificationID string
	if err := tx.QueryRow(ctx, `select id from notification_events where idempotency_key = $1`, params.IdempotencyKey).Scan(&notificationID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `update reminder_events set notification_event_id = $1 where id = $2`, notificationID, reminderID); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) commitmentReminderRecipients(ctx context.Context, tx pgx.Tx, workspaceID string, ownerPersonID sql.NullString) ([]string, error) {
	if tx == nil {
		return nil, nil
	}
	if ownerPersonID.Valid {
		var email string
		err := tx.QueryRow(ctx, `
			select p.email
			from workspace_members wm
			join people p on p.id = wm.person_id
			where wm.workspace_id = $1
			  and wm.person_id = $2
			  and wm.removed_at is null
		`, workspaceID, ownerPersonID.String).Scan(&email)
		if err == nil {
			normalized := normalizeEmail(email)
			if normalized != "" {
				return []string{normalized}, nil
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	rows, err := tx.Query(ctx, `
		select p.email
		from workspace_members wm
		join people p on p.id = wm.person_id
		where wm.workspace_id = $1
		  and wm.role = 'owner'
		  and wm.removed_at is null
		order by p.email
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recipients := make([]string, 0)
	seen := map[string]struct{}{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		normalized := normalizeEmail(email)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		recipients = append(recipients, normalized)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return recipients, nil
}

func commitmentReminderCopy(title string, dueAt time.Time) (string, string, string) {
	due := dueAt.UTC().Format(time.RFC3339)
	subject := fmt.Sprintf("Commitment due: %s", title)
	preview := fmt.Sprintf("%s due %s", title, due)
	body := fmt.Sprintf("Commitment %q is due at %s.", title, due)
	return subject, preview, body
}
