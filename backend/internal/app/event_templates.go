package app

import (
	"net/http"
	"time"
)

type eventTemplateDTO struct {
	ID                string `json:"id"`
	WorkspaceID       string `json:"workspaceId"`
	Name              string `json:"name"`
	Title             string `json:"title"`
	PublicDescription string `json:"publicDescription"`
	LocationDisplay   string `json:"locationDisplay"`
	TicketAllocation  int    `json:"ticketAllocation"`
	PricingMode       string `json:"pricingMode"`
	TicketPriceCents  int    `json:"ticketPriceCents"`
	TicketCurrency    string `json:"ticketCurrency"`
	PrivateNotes      string `json:"privateNotes"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type eventTemplateRow struct {
	ID                string
	WorkspaceID       string
	Name              string
	Title             string
	PublicDescription string
	LocationDisplay   string
	TicketAllocation  int
	PricingMode       string
	TicketPriceCents  int
	TicketCurrency    string
	PrivateNotes      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (a *App) handleListEventTemplates(w http.ResponseWriter, r *http.Request) {
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
		select id, workspace_id, name, title, public_description, location_display,
		       ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
		       private_notes, created_at, updated_at
		from event_templates
		where workspace_id = $1
		order by lower(name), created_at, id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load event templates")
		return
	}
	defer rows.Close()

	templates := make([]eventTemplateDTO, 0)
	for rows.Next() {
		var row eventTemplateRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.Name, &row.Title, &row.PublicDescription, &row.LocationDisplay, &row.TicketAllocation, &row.PricingMode, &row.TicketPriceCents, &row.TicketCurrency, &row.PrivateNotes, &row.CreatedAt, &row.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load event templates")
			return
		}
		templates = append(templates, eventTemplateDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load event templates")
		return
	}

	writeJSON(w, http.StatusOK, templates)
}

func eventTemplateDTOFromRow(row eventTemplateRow) eventTemplateDTO {
	return eventTemplateDTO{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Name:              row.Name,
		Title:             row.Title,
		PublicDescription: row.PublicDescription,
		LocationDisplay:   row.LocationDisplay,
		TicketAllocation:  row.TicketAllocation,
		PricingMode:       row.PricingMode,
		TicketPriceCents:  row.TicketPriceCents,
		TicketCurrency:    row.TicketCurrency,
		PrivateNotes:      row.PrivateNotes,
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         row.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
