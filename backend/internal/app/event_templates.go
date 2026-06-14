package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
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

type createEventTemplateRequest struct {
	Name              string `json:"name"`
	Title             string `json:"title"`
	PublicDescription string `json:"publicDescription"`
	LocationDisplay   string `json:"locationDisplay"`
	TicketAllocation  int    `json:"ticketAllocation"`
	PricingMode       string `json:"pricingMode"`
	TicketPriceCents  int    `json:"ticketPriceCents"`
	TicketCurrency    string `json:"ticketCurrency"`
	PrivateNotes      string `json:"privateNotes"`
}

type updateEventTemplateRequest struct {
	Name              *string `json:"name"`
	Title             *string `json:"title"`
	PublicDescription *string `json:"publicDescription"`
	LocationDisplay   *string `json:"locationDisplay"`
	TicketAllocation  *int    `json:"ticketAllocation"`
	PricingMode       *string `json:"pricingMode"`
	TicketPriceCents  *int    `json:"ticketPriceCents"`
	TicketCurrency    *string `json:"ticketCurrency"`
	PrivateNotes      *string `json:"privateNotes"`
}

type applyEventTemplateRequest struct {
	TemplateID string `json:"templateId"`
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

func (a *App) handleCreateEventTemplate(w http.ResponseWriter, r *http.Request) {
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

	var req createEventTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	name, err := normalizeEventTemplateRequiredString(req.Name, "name")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	title, err := normalizeEventTemplateRequiredString(req.Title, "title")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	publicDescription, err := normalizeEventTemplateText(req.PublicDescription, "publicDescription")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	locationDisplay, err := normalizeEventTemplateText(req.LocationDisplay, "locationDisplay")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	privateNotes, err := normalizeEventTemplateText(req.PrivateNotes, "privateNotes")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.TicketAllocation < 0 {
		writeError(w, http.StatusBadRequest, "ticketAllocation must be non-negative")
		return
	}
	pricingMode, ticketPriceCents, ticketCurrency, err := normalizeEventPricing(req.PricingMode, req.TicketPriceCents, req.TicketCurrency)
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

	var row eventTemplateRow
	if err := tx.QueryRow(r.Context(), `
		insert into event_templates (
			workspace_id, name, title, public_description, location_display,
			ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
			private_notes, created_by_person_id
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		returning id, workspace_id, name, title, public_description, location_display,
		          ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
		          private_notes, created_at, updated_at
	`, workspaceID, name, title, publicDescription, locationDisplay, req.TicketAllocation, pricingMode, ticketPriceCents, ticketCurrency, privateNotes, actorID).Scan(
		&row.ID, &row.WorkspaceID, &row.Name, &row.Title, &row.PublicDescription, &row.LocationDisplay,
		&row.TicketAllocation, &row.PricingMode, &row.TicketPriceCents, &row.TicketCurrency,
		&row.PrivateNotes, &row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "template name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create event template")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actorID, "event_template.created", "event_template", row.ID, map[string]any{
		"templateId":  row.ID,
		"workspaceId": workspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event template")
		return
	}

	writeJSON(w, http.StatusOK, eventTemplateDTOFromRow(row))
}

func (a *App) handleUpdateEventTemplate(w http.ResponseWriter, r *http.Request) {
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

	var req updateEventTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Name == nil && req.Title == nil && req.PublicDescription == nil && req.LocationDisplay == nil && req.TicketAllocation == nil && req.PricingMode == nil && req.TicketPriceCents == nil && req.TicketCurrency == nil && req.PrivateNotes == nil {
		writeError(w, http.StatusBadRequest, "no changes provided")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var current eventTemplateRow
	if err := tx.QueryRow(r.Context(), `
		select id, workspace_id, name, title, public_description, location_display,
		       ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
		       private_notes, created_at, updated_at
		from event_templates
		where workspace_id = $1
		  and id = $2
		for update
	`, workspaceID, r.PathValue("templateID")).Scan(
		&current.ID, &current.WorkspaceID, &current.Name, &current.Title, &current.PublicDescription, &current.LocationDisplay,
		&current.TicketAllocation, &current.PricingMode, &current.TicketPriceCents, &current.TicketCurrency,
		&current.PrivateNotes, &current.CreatedAt, &current.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event template")
		return
	}

	newRow := current
	changed := false

	if req.Name != nil {
		name, err := normalizeEventTemplateRequiredString(*req.Name, "name")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if name != current.Name {
			newRow.Name = name
			changed = true
		}
	}
	if req.Title != nil {
		title, err := normalizeEventTemplateRequiredString(*req.Title, "title")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if title != current.Title {
			newRow.Title = title
			changed = true
		}
	}
	if req.PublicDescription != nil {
		value, err := normalizeEventTemplateText(*req.PublicDescription, "publicDescription")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if value != current.PublicDescription {
			newRow.PublicDescription = value
			changed = true
		}
	}
	if req.LocationDisplay != nil {
		value, err := normalizeEventTemplateText(*req.LocationDisplay, "locationDisplay")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if value != current.LocationDisplay {
			newRow.LocationDisplay = value
			changed = true
		}
	}
	if req.PrivateNotes != nil {
		value, err := normalizeEventTemplateText(*req.PrivateNotes, "privateNotes")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if value != current.PrivateNotes {
			newRow.PrivateNotes = value
			changed = true
		}
	}
	if req.TicketAllocation != nil {
		if *req.TicketAllocation < 0 {
			writeError(w, http.StatusBadRequest, "ticketAllocation must be non-negative")
			return
		}
		if *req.TicketAllocation != current.TicketAllocation {
			newRow.TicketAllocation = *req.TicketAllocation
			changed = true
		}
	}
	if req.PricingMode != nil || req.TicketPriceCents != nil || req.TicketCurrency != nil {
		candidateMode := current.PricingMode
		candidatePriceCents := current.TicketPriceCents
		candidateCurrency := current.TicketCurrency
		if req.PricingMode != nil {
			candidateMode = *req.PricingMode
		}
		if req.TicketPriceCents != nil {
			candidatePriceCents = *req.TicketPriceCents
		}
		if req.TicketCurrency != nil {
			candidateCurrency = *req.TicketCurrency
		}
		normalizedMode, normalizedPriceCents, normalizedCurrency, pricingErr := normalizeEventPricing(candidateMode, candidatePriceCents, candidateCurrency)
		if pricingErr != nil {
			writeError(w, http.StatusBadRequest, pricingErr.Error())
			return
		}
		if normalizedMode != current.PricingMode || normalizedPriceCents != current.TicketPriceCents || normalizedCurrency != current.TicketCurrency {
			newRow.PricingMode = normalizedMode
			newRow.TicketPriceCents = normalizedPriceCents
			newRow.TicketCurrency = normalizedCurrency
			changed = true
		}
	}

	if !changed {
		writeError(w, http.StatusBadRequest, "no changes provided")
		return
	}

	if err := tx.QueryRow(r.Context(), `
		update event_templates
		set name = $3,
		    title = $4,
		    public_description = $5,
		    location_display = $6,
		    ticket_allocation = $7,
		    pricing_mode = $8,
		    ticket_price_cents = $9,
		    ticket_currency = $10,
		    private_notes = $11,
		    updated_at = now()
		where workspace_id = $1
		  and id = $2
		returning id, workspace_id, name, title, public_description, location_display,
		          ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
		          private_notes, created_at, updated_at
	`, workspaceID, current.ID, newRow.Name, newRow.Title, newRow.PublicDescription, newRow.LocationDisplay, newRow.TicketAllocation, newRow.PricingMode, newRow.TicketPriceCents, newRow.TicketCurrency, newRow.PrivateNotes).Scan(
		&newRow.ID, &newRow.WorkspaceID, &newRow.Name, &newRow.Title, &newRow.PublicDescription, &newRow.LocationDisplay,
		&newRow.TicketAllocation, &newRow.PricingMode, &newRow.TicketPriceCents, &newRow.TicketCurrency,
		&newRow.PrivateNotes, &newRow.CreatedAt, &newRow.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "template name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update event template")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actorID, "event_template.updated", "event_template", newRow.ID, map[string]any{
		"templateId":  newRow.ID,
		"workspaceId": workspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event template")
		return
	}

	writeJSON(w, http.StatusOK, eventTemplateDTOFromRow(newRow))
}

func (a *App) handleDeleteEventTemplate(w http.ResponseWriter, r *http.Request) {
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

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var deleted eventTemplateRow
	if err := tx.QueryRow(r.Context(), `
		delete from event_templates
		where workspace_id = $1
		  and id = $2
		returning id, workspace_id, name, title, public_description, location_display,
		          ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
		          private_notes, created_at, updated_at
	`, workspaceID, r.PathValue("templateID")).Scan(
		&deleted.ID, &deleted.WorkspaceID, &deleted.Name, &deleted.Title, &deleted.PublicDescription, &deleted.LocationDisplay,
		&deleted.TicketAllocation, &deleted.PricingMode, &deleted.TicketPriceCents, &deleted.TicketCurrency,
		&deleted.PrivateNotes, &deleted.CreatedAt, &deleted.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete event template")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actorID, "event_template.deleted", "event_template", deleted.ID, map[string]any{
		"templateId":  deleted.ID,
		"workspaceId": workspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event template")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleApplyEventTemplate(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	var req applyEventTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	templateID := strings.TrimSpace(req.TemplateID)
	if templateID == "" {
		writeError(w, http.StatusBadRequest, "templateId is required")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	eventID := r.PathValue("eventID")
	event, err := a.loadEventDetailsForUpdate(r.Context(), tx, eventID)
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
	if event.Status != "draft" {
		writeError(w, http.StatusConflict, "event must be draft")
		return
	}

	var template eventTemplateRow
	if err := tx.QueryRow(r.Context(), `
		select id, workspace_id, name, title, public_description, location_display,
		       ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency,
		       private_notes, created_at, updated_at
		from event_templates
		where workspace_id = $1
		  and id = $2
	`, event.WorkspaceID, templateID).Scan(&template.ID, &template.WorkspaceID, &template.Name, &template.Title, &template.PublicDescription, &template.LocationDisplay, &template.TicketAllocation, &template.PricingMode, &template.TicketPriceCents, &template.TicketCurrency, &template.PrivateNotes, &template.CreatedAt, &template.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event template")
		return
	}

	if event.ReservedCount > 0 {
		if template.PricingMode != event.PricingMode || template.TicketPriceCents != event.TicketPriceCents || template.TicketCurrency != event.TicketCurrency {
			writeError(w, http.StatusConflict, "pricing cannot change after tickets exist")
			return
		}
		if template.TicketAllocation < event.ReservedCount {
			writeError(w, http.StatusConflict, "ticket allocation cannot go below reserved tickets")
			return
		}
	}

	changed := template.Title != event.Title || template.PublicDescription != event.PublicDescription || template.LocationDisplay != event.LocationDisplay || template.TicketAllocation != event.TicketAllocation || template.PricingMode != event.PricingMode || template.TicketPriceCents != event.TicketPriceCents || template.TicketCurrency != event.TicketCurrency
	if changed {
		if _, err := tx.Exec(r.Context(), `
			update events
			set title = $2,
			    public_description = $3,
			    location_display = $4,
			    ticket_allocation = $5,
			    pricing_mode = $6,
			    ticket_price_cents = $7,
			    ticket_currency = $8,
			    updated_at = now()
			where id = $1
		`, event.ID, template.Title, template.PublicDescription, template.LocationDisplay, template.TicketAllocation, template.PricingMode, template.TicketPriceCents, template.TicketCurrency); err != nil {
			writeError(w, http.StatusInternalServerError, "could not update event")
			return
		}
		event.Title = template.Title
		event.PublicDescription = template.PublicDescription
		event.LocationDisplay = template.LocationDisplay
		event.TicketAllocation = template.TicketAllocation
		event.PricingMode = template.PricingMode
		event.TicketPriceCents = template.TicketPriceCents
		event.TicketCurrency = template.TicketCurrency
		if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actorID, "event_template.applied", "event", event.ID, map[string]any{
			"templateId":  template.ID,
			"eventId":     event.ID,
			"workspaceId": event.WorkspaceID,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "could not record audit")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event")
		return
	}

	writeJSON(w, http.StatusOK, a.eventDTOFromRow(event))
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

func normalizeEventTemplateRequiredString(value, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return trimmed, nil
}

func normalizeEventTemplateText(value, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if utf8.RuneCountInString(trimmed) > 2000 {
		return "", fmt.Errorf("%s must be 2000 characters or fewer", field)
	}
	return trimmed, nil
}
