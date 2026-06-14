package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type commitmentDTO struct {
	ID                  string  `json:"id"`
	WorkspaceID         string  `json:"workspaceId"`
	EventID             *string `json:"eventId,omitempty"`
	ContactID           *string `json:"contactId,omitempty"`
	Title               string  `json:"title"`
	Description         string  `json:"description"`
	DueAt               *string `json:"dueAt,omitempty"`
	Status              string  `json:"status"`
	OwnerPersonID       *string `json:"ownerPersonId,omitempty"`
	CreatedByPersonID   string  `json:"createdByPersonId"`
	CompletedAt         *string `json:"completedAt,omitempty"`
	CompletedByPersonID *string `json:"completedByPersonId,omitempty"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
}

type createCommitmentRequest struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	DueAt         *string `json:"dueAt"`
	EventID       *string `json:"eventId"`
	ContactID     *string `json:"contactId"`
	OwnerPersonID *string `json:"ownerPersonId"`
}

type updateCommitmentRequest struct {
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	DueAt         *string `json:"dueAt"`
	EventID       *string `json:"eventId"`
	ContactID     *string `json:"contactId"`
	OwnerPersonID *string `json:"ownerPersonId"`
	Status        *string `json:"status"`
	ClearDueAt    bool    `json:"clearDueAt"`
	ClearEvent    bool    `json:"clearEvent"`
	ClearContact  bool    `json:"clearContact"`
	ClearOwner    bool    `json:"clearOwner"`
}

type commitmentRow struct {
	ID                  string
	WorkspaceID         string
	EventID             sql.NullString
	ContactID           sql.NullString
	Title               string
	Description         string
	DueAt               sql.NullTime
	Status              string
	OwnerPersonID       sql.NullString
	CreatedByPersonID   string
	CompletedAt         sql.NullTime
	CompletedByPersonID sql.NullString
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type commitmentRowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (a *App) handleListWorkspaceCommitments(w http.ResponseWriter, r *http.Request) {
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
		select id, workspace_id, event_id, contact_id, title, description, due_at, status,
		       owner_person_id, created_by_person_id, completed_at, completed_by_person_id, created_at, updated_at
		from commitments
		where workspace_id = $1
		order by case status when 'open' then 0 when 'done' then 1 when 'cancelled' then 2 else 3 end,
		         due_at nulls last, created_at desc, id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load commitments")
		return
	}
	defer rows.Close()

	items := make([]commitmentDTO, 0)
	for rows.Next() {
		var row commitmentRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.EventID, &row.ContactID, &row.Title, &row.Description, &row.DueAt, &row.Status, &row.OwnerPersonID, &row.CreatedByPersonID, &row.CompletedAt, &row.CompletedByPersonID, &row.CreatedAt, &row.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load commitments")
			return
		}
		items = append(items, commitmentDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load commitments")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleListEventCommitments(w http.ResponseWriter, r *http.Request) {
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
		select id, workspace_id, event_id, contact_id, title, description, due_at, status,
		       owner_person_id, created_by_person_id, completed_at, completed_by_person_id, created_at, updated_at
		from commitments
		where event_id = $1
		order by case status when 'open' then 0 when 'done' then 1 when 'cancelled' then 2 else 3 end,
		         due_at nulls last, created_at desc, id
	`, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load commitments")
		return
	}
	defer rows.Close()

	items := make([]commitmentDTO, 0)
	for rows.Next() {
		var row commitmentRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.EventID, &row.ContactID, &row.Title, &row.Description, &row.DueAt, &row.Status, &row.OwnerPersonID, &row.CreatedByPersonID, &row.CompletedAt, &row.CompletedByPersonID, &row.CreatedAt, &row.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load commitments")
			return
		}
		items = append(items, commitmentDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load commitments")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleCreateCommitment(w http.ResponseWriter, r *http.Request) {
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

	var req createCommitmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	description := strings.TrimSpace(req.Description)
	dueAt, err := parseOptionalRFC3339Time(req.DueAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid dueAt")
		return
	}
	eventID, err := normalizeOptionalID(req.EventID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	contactID, err := normalizeOptionalID(req.ContactID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ownerPersonID, err := normalizeOptionalID(req.OwnerPersonID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if err := ensureCommitmentWorkspaceEvent(r.Context(), tx, workspaceID, eventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "eventId must belong to workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not validate event")
		return
	}
	if err := ensureCommitmentWorkspaceContact(r.Context(), tx, workspaceID, contactID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "contactId must belong to workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not validate contact")
		return
	}
	if err := ensureCommitmentWorkspaceOwner(r.Context(), tx, workspaceID, ownerPersonID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "ownerPersonId must belong to workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not validate owner")
		return
	}

	var row commitmentRow
	if err := tx.QueryRow(r.Context(), `
		insert into commitments (
			workspace_id, event_id, contact_id, title, description, due_at, status, owner_person_id, created_by_person_id, completed_at, completed_by_person_id
		)
		values ($1, $2, $3, $4, $5, $6, 'open', $7, $8, null, null)
		returning id, workspace_id, event_id, contact_id, title, description, due_at, status,
		          owner_person_id, created_by_person_id, completed_at, completed_by_person_id, created_at, updated_at
	`, workspaceID, nullableStringArg(eventID), nullableStringArg(contactID), title, description, dueAt, nullableStringArg(ownerPersonID), actorID).Scan(
		&row.ID, &row.WorkspaceID, &row.EventID, &row.ContactID, &row.Title, &row.Description, &row.DueAt, &row.Status,
		&row.OwnerPersonID, &row.CreatedByPersonID, &row.CompletedAt, &row.CompletedByPersonID, &row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create commitment")
		return
	}

	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "commitment.created", "commitment", row.ID, map[string]any{
		"commitmentId": row.ID,
		"workspaceId":  workspaceID,
		"status":       row.Status,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save commitment")
		return
	}

	writeJSON(w, http.StatusOK, commitmentDTOFromRow(row))
}

func (a *App) handleUpdateCommitment(w http.ResponseWriter, r *http.Request) {
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

	var req updateCommitmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ClearDueAt && req.DueAt != nil {
		writeError(w, http.StatusBadRequest, "clearDueAt conflicts with dueAt")
		return
	}
	if req.ClearEvent && req.EventID != nil {
		writeError(w, http.StatusBadRequest, "clearEvent conflicts with eventId")
		return
	}
	if req.ClearContact && req.ContactID != nil {
		writeError(w, http.StatusBadRequest, "clearContact conflicts with contactId")
		return
	}
	if req.ClearOwner && req.OwnerPersonID != nil {
		writeError(w, http.StatusBadRequest, "clearOwner conflicts with ownerPersonId")
		return
	}

	var requestedTitle *string
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		requestedTitle = &title
	}
	var requestedDescription *string
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		requestedDescription = &description
	}
	var requestedDueAt *time.Time
	var err error
	if !req.ClearDueAt && req.DueAt != nil {
		requestedDueAt, err = parseOptionalRFC3339Time(req.DueAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueAt")
			return
		}
	}
	requestedEventID, err := normalizeOptionalID(req.EventID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	requestedContactID, err := normalizeOptionalID(req.ContactID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	requestedOwnerPersonID, err := normalizeOptionalID(req.OwnerPersonID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var requestedStatus *string
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		switch status {
		case "open", "done", "cancelled":
			requestedStatus = &status
		default:
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	current, err := loadCommitmentRow(r.Context(), tx, workspaceID, r.PathValue("commitmentID"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "commitment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load commitment")
		return
	}

	if err := ensureCommitmentWorkspaceEvent(r.Context(), tx, workspaceID, requestedEventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "eventId must belong to workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not validate event")
		return
	}
	if err := ensureCommitmentWorkspaceContact(r.Context(), tx, workspaceID, requestedContactID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "contactId must belong to workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not validate contact")
		return
	}
	if err := ensureCommitmentWorkspaceOwner(r.Context(), tx, workspaceID, requestedOwnerPersonID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "ownerPersonId must belong to workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not validate owner")
		return
	}

	newRow := current
	changed := false
	if requestedTitle != nil && *requestedTitle != current.Title {
		newRow.Title = *requestedTitle
		changed = true
	}
	if requestedDescription != nil && *requestedDescription != current.Description {
		newRow.Description = *requestedDescription
		changed = true
	}
	if req.ClearDueAt {
		if current.DueAt.Valid {
			newRow.DueAt = sql.NullTime{}
			changed = true
		}
	} else if requestedDueAt != nil {
		if !current.DueAt.Valid || !current.DueAt.Time.Equal(*requestedDueAt) {
			newRow.DueAt = sql.NullTime{Time: *requestedDueAt, Valid: true}
			changed = true
		}
	}
	if req.ClearEvent {
		if current.EventID.Valid {
			newRow.EventID = sql.NullString{}
			changed = true
		}
	} else if requestedEventID != nil {
		if !current.EventID.Valid || current.EventID.String != *requestedEventID {
			newRow.EventID = sql.NullString{String: *requestedEventID, Valid: true}
			changed = true
		}
	}
	if req.ClearContact {
		if current.ContactID.Valid {
			newRow.ContactID = sql.NullString{}
			changed = true
		}
	} else if requestedContactID != nil {
		if !current.ContactID.Valid || current.ContactID.String != *requestedContactID {
			newRow.ContactID = sql.NullString{String: *requestedContactID, Valid: true}
			changed = true
		}
	}
	if req.ClearOwner {
		if current.OwnerPersonID.Valid {
			newRow.OwnerPersonID = sql.NullString{}
			changed = true
		}
	} else if requestedOwnerPersonID != nil {
		if !current.OwnerPersonID.Valid || current.OwnerPersonID.String != *requestedOwnerPersonID {
			newRow.OwnerPersonID = sql.NullString{String: *requestedOwnerPersonID, Valid: true}
			changed = true
		}
	}
	if requestedStatus != nil {
		switch *requestedStatus {
		case "done":
			if current.Status != "done" {
				newRow.Status = "done"
				newRow.CompletedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
				newRow.CompletedByPersonID = sql.NullString{String: actorID, Valid: true}
				changed = true
			}
		case "open", "cancelled":
			if current.Status != *requestedStatus {
				newRow.Status = *requestedStatus
				changed = true
			}
			if current.Status == "done" {
				newRow.CompletedAt = sql.NullTime{}
				newRow.CompletedByPersonID = sql.NullString{}
				changed = true
			}
		}
	}

	if !changed {
		writeJSON(w, http.StatusOK, commitmentDTOFromRow(current))
		return
	}

	if err := tx.QueryRow(r.Context(), `
		update commitments
		set event_id = $3,
		    contact_id = $4,
		    title = $5,
		    description = $6,
		    due_at = $7,
		    status = $8,
		    owner_person_id = $9,
		    completed_at = $10,
		    completed_by_person_id = $11,
		    updated_at = now()
		where workspace_id = $1
		  and id = $2
		returning id, workspace_id, event_id, contact_id, title, description, due_at, status,
		          owner_person_id, created_by_person_id, completed_at, completed_by_person_id, created_at, updated_at
	`, workspaceID, current.ID, newRow.EventID, newRow.ContactID, newRow.Title, newRow.Description, newRow.DueAt, newRow.Status, newRow.OwnerPersonID, newRow.CompletedAt, newRow.CompletedByPersonID).Scan(
		&newRow.ID, &newRow.WorkspaceID, &newRow.EventID, &newRow.ContactID, &newRow.Title, &newRow.Description, &newRow.DueAt, &newRow.Status,
		&newRow.OwnerPersonID, &newRow.CreatedByPersonID, &newRow.CompletedAt, &newRow.CompletedByPersonID, &newRow.CreatedAt, &newRow.UpdatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update commitment")
		return
	}

	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "commitment.updated", "commitment", newRow.ID, map[string]any{
		"commitmentId": newRow.ID,
		"workspaceId":  workspaceID,
		"status":       newRow.Status,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save commitment")
		return
	}

	writeJSON(w, http.StatusOK, commitmentDTOFromRow(newRow))
}

func loadCommitmentRow(ctx context.Context, q commitmentRowQuerier, workspaceID, commitmentID string) (commitmentRow, error) {
	var row commitmentRow
	if err := q.QueryRow(ctx, `
		select id, workspace_id, event_id, contact_id, title, description, due_at, status,
		       owner_person_id, created_by_person_id, completed_at, completed_by_person_id, created_at, updated_at
		from commitments
		where workspace_id = $1
		  and id = $2
		for update
	`, workspaceID, commitmentID).Scan(&row.ID, &row.WorkspaceID, &row.EventID, &row.ContactID, &row.Title, &row.Description, &row.DueAt, &row.Status, &row.OwnerPersonID, &row.CreatedByPersonID, &row.CompletedAt, &row.CompletedByPersonID, &row.CreatedAt, &row.UpdatedAt); err != nil {
		return commitmentRow{}, err
	}
	return row, nil
}

func commitmentDTOFromRow(row commitmentRow) commitmentDTO {
	dto := commitmentDTO{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Title:             row.Title,
		Description:       row.Description,
		Status:            row.Status,
		CreatedByPersonID: row.CreatedByPersonID,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if row.EventID.Valid {
		eventID := row.EventID.String
		dto.EventID = &eventID
	}
	if row.ContactID.Valid {
		contactID := row.ContactID.String
		dto.ContactID = &contactID
	}
	if row.DueAt.Valid {
		dueAt := row.DueAt.Time.UTC().Format(time.RFC3339Nano)
		dto.DueAt = &dueAt
	}
	if row.OwnerPersonID.Valid {
		ownerPersonID := row.OwnerPersonID.String
		dto.OwnerPersonID = &ownerPersonID
	}
	if row.CompletedAt.Valid {
		completedAt := row.CompletedAt.Time.UTC().Format(time.RFC3339Nano)
		dto.CompletedAt = &completedAt
	}
	if row.CompletedByPersonID.Valid {
		completedByPersonID := row.CompletedByPersonID.String
		dto.CompletedByPersonID = &completedByPersonID
	}
	return dto
}

func normalizeOptionalID(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, fmt.Errorf("id is required")
	}
	return &trimmed, nil
}

func nullableStringArg(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func ensureCommitmentWorkspaceEvent(ctx context.Context, q commitmentRowQuerier, workspaceID string, eventID *string) error {
	if eventID == nil {
		return nil
	}
	var id string
	return q.QueryRow(ctx, `
		select id
		from events
		where id = $1
		  and workspace_id = $2
	`, *eventID, workspaceID).Scan(&id)
}

func ensureCommitmentWorkspaceContact(ctx context.Context, q commitmentRowQuerier, workspaceID string, contactID *string) error {
	if contactID == nil {
		return nil
	}
	var id string
	return q.QueryRow(ctx, `
		select id
		from contacts
		where id = $1
		  and workspace_id = $2
	`, *contactID, workspaceID).Scan(&id)
}

func ensureCommitmentWorkspaceOwner(ctx context.Context, q commitmentRowQuerier, workspaceID string, ownerPersonID *string) error {
	if ownerPersonID == nil {
		return nil
	}
	var id string
	return q.QueryRow(ctx, `
		select person_id
		from workspace_members
		where workspace_id = $1
		  and person_id = $2
		  and removed_at is null
	`, workspaceID, *ownerPersonID).Scan(&id)
}
