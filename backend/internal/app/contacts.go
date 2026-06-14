package app

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
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
