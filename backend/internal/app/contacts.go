package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type contactDTO struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspaceId"`
	DisplayName string   `json:"displayName"`
	Email       *string  `json:"email,omitempty"`
	Phone       *string  `json:"phone,omitempty"`
	Notes       string   `json:"notes"`
	Tags        []string `json:"tags"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type createContactRequest struct {
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	Notes       string   `json:"notes"`
	Tags        []string `json:"tags"`
}

type updateContactRequest struct {
	DisplayName *string  `json:"displayName"`
	Email       *string  `json:"email"`
	Phone       *string  `json:"phone"`
	Notes       *string  `json:"notes"`
	Tags        []string `json:"tags"`
	ClearEmail  bool     `json:"clearEmail"`
	ClearPhone  bool     `json:"clearPhone"`
}

type contactRow struct {
	ID          string
	WorkspaceID string
	DisplayName string
	Email       sql.NullString
	Phone       sql.NullString
	Notes       string
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (a *App) handleListContacts(w http.ResponseWriter, r *http.Request) {
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
		select id, workspace_id, display_name, email, phone, notes, tags, created_at, updated_at
		from contacts
		where workspace_id = $1
		order by lower(display_name), created_at, id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load contacts")
		return
	}
	defer rows.Close()

	contacts := make([]contactDTO, 0)
	for rows.Next() {
		var row contactRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.DisplayName, &row.Email, &row.Phone, &row.Notes, &row.Tags, &row.CreatedAt, &row.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load contacts")
			return
		}
		contacts = append(contacts, contactDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load contacts")
		return
	}

	writeJSON(w, http.StatusOK, contacts)
}

func (a *App) handleCreateContact(w http.ResponseWriter, r *http.Request) {
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

	var req createContactRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		writeError(w, http.StatusBadRequest, "displayName is required")
		return
	}
	notes := strings.TrimSpace(req.Notes)
	if utf8.RuneCountInString(notes) > 2000 {
		writeError(w, http.StatusBadRequest, "notes must be 2000 characters or fewer")
		return
	}
	tags, err := normalizeContactTags(req.Tags)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	email, err := normalizeContactEmail(req.Email, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	phone, err := normalizeContactPhone(req.Phone, true)
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

	var contact contactRow
	if err := tx.QueryRow(r.Context(), `
		insert into contacts (workspace_id, display_name, email, phone, notes, tags, created_by_person_id)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, workspace_id, display_name, email, phone, notes, tags, created_at, updated_at
	`, workspaceID, displayName, email, phone, notes, tags, actorID).Scan(&contact.ID, &contact.WorkspaceID, &contact.DisplayName, &contact.Email, &contact.Phone, &contact.Notes, &contact.Tags, &contact.CreatedAt, &contact.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "contact email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create contact")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "contact.created", "contact", contact.ID, map[string]any{
		"contactId":   contact.ID,
		"workspaceId": workspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save contact")
		return
	}

	writeJSON(w, http.StatusOK, contactDTOFromRow(contact))
}

func (a *App) handleUpdateContact(w http.ResponseWriter, r *http.Request) {
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

	var req updateContactRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ClearEmail && req.Email != nil {
		writeError(w, http.StatusBadRequest, "clearEmail conflicts with email")
		return
	}
	if req.ClearPhone && req.Phone != nil {
		writeError(w, http.StatusBadRequest, "clearPhone conflicts with phone")
		return
	}

	var requestedDisplayName *string
	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		if displayName == "" {
			writeError(w, http.StatusBadRequest, "displayName is required")
			return
		}
		requestedDisplayName = &displayName
	}
	var requestedNotes *string
	if req.Notes != nil {
		notes := strings.TrimSpace(*req.Notes)
		if utf8.RuneCountInString(notes) > 2000 {
			writeError(w, http.StatusBadRequest, "notes must be 2000 characters or fewer")
			return
		}
		requestedNotes = &notes
	}
	var requestedTags []string
	var tagsProvided bool
	if req.Tags != nil {
		var err error
		requestedTags, err = normalizeContactTags(req.Tags)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		tagsProvided = true
	}
	var requestedEmail *string
	if req.ClearEmail {
		requestedEmail = nil
	} else if req.Email != nil {
		email, err := normalizeContactEmail(*req.Email, false)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		requestedEmail = email
	}
	var requestedPhone *string
	if req.ClearPhone {
		requestedPhone = nil
	} else if req.Phone != nil {
		phone, err := normalizeContactPhone(*req.Phone, false)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		requestedPhone = phone
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var current contactRow
	if err := tx.QueryRow(r.Context(), `
		select id, workspace_id, display_name, email, phone, notes, tags, created_at, updated_at
		from contacts
		where workspace_id = $1
		  and id = $2
		for update
	`, workspaceID, r.PathValue("contactID")).Scan(&current.ID, &current.WorkspaceID, &current.DisplayName, &current.Email, &current.Phone, &current.Notes, &current.Tags, &current.CreatedAt, &current.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "contact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load contact")
		return
	}

	newRow := current
	changed := false
	if requestedDisplayName != nil && *requestedDisplayName != current.DisplayName {
		newRow.DisplayName = *requestedDisplayName
		changed = true
	}
	if requestedNotes != nil && *requestedNotes != current.Notes {
		newRow.Notes = *requestedNotes
		changed = true
	}
	if req.ClearEmail {
		if current.Email.Valid {
			changed = true
		}
		newRow.Email = sql.NullString{}
	} else if requestedEmail != nil {
		candidate := sql.NullString{String: *requestedEmail, Valid: true}
		if !current.Email.Valid || current.Email.String != candidate.String {
			changed = true
		}
		newRow.Email = candidate
	}
	if req.ClearPhone {
		if current.Phone.Valid {
			changed = true
		}
		newRow.Phone = sql.NullString{}
	} else if requestedPhone != nil {
		candidate := sql.NullString{String: *requestedPhone, Valid: true}
		if !current.Phone.Valid || current.Phone.String != candidate.String {
			changed = true
		}
		newRow.Phone = candidate
	}
	if tagsProvided && !contactTagsEqual(current.Tags, requestedTags) {
		newRow.Tags = requestedTags
		changed = true
	}

	if !changed {
		writeJSON(w, http.StatusOK, contactDTOFromRow(current))
		return
	}

	if err := tx.QueryRow(r.Context(), `
		update contacts
		set display_name = $3,
		    email = $4,
		    phone = $5,
		    notes = $6,
		    tags = $7,
		    updated_at = now()
		where workspace_id = $1
		  and id = $2
		returning id, workspace_id, display_name, email, phone, notes, tags, created_at, updated_at
	`, workspaceID, current.ID, newRow.DisplayName, newRow.Email, newRow.Phone, newRow.Notes, newRow.Tags).Scan(&newRow.ID, &newRow.WorkspaceID, &newRow.DisplayName, &newRow.Email, &newRow.Phone, &newRow.Notes, &newRow.Tags, &newRow.CreatedAt, &newRow.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "contact email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update contact")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "contact.updated", "contact", newRow.ID, map[string]any{
		"contactId":   newRow.ID,
		"workspaceId": workspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save contact")
		return
	}

	writeJSON(w, http.StatusOK, contactDTOFromRow(newRow))
}

func contactDTOFromRow(row contactRow) contactDTO {
	dto := contactDTO{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		DisplayName: row.DisplayName,
		Notes:       row.Notes,
		Tags:        row.Tags,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if dto.Tags == nil {
		dto.Tags = []string{}
	}
	if row.Email.Valid {
		email := strings.TrimSpace(row.Email.String)
		if email != "" {
			dto.Email = &email
		}
	}
	if row.Phone.Valid {
		phone := strings.TrimSpace(row.Phone.String)
		if phone != "" {
			dto.Phone = &phone
		}
	}
	return dto
}

func normalizeContactEmail(value string, emptyIsNil bool) (*string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if emptyIsNil {
			return nil, nil
		}
		return nil, fmt.Errorf("email is required")
	}
	trimmed = strings.ToLower(trimmed)
	if !strings.Contains(trimmed, "@") {
		return nil, fmt.Errorf("invalid email")
	}
	return &trimmed, nil
}

func normalizeContactPhone(value string, emptyIsNil bool) (*string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if emptyIsNil {
			return nil, nil
		}
		return nil, fmt.Errorf("phone is required")
	}
	return &trimmed, nil
}

func normalizeContactTags(tags []string) ([]string, error) {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		if utf8.RuneCountInString(trimmed) > 40 {
			return nil, fmt.Errorf("tags must be 40 characters or fewer")
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) > 20 {
		return nil, fmt.Errorf("tags must be 20 items or fewer")
	}
	if normalized == nil {
		normalized = []string{}
	}
	return normalized, nil
}

func contactTagsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
