package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type eventDTO struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspaceId"`
	Title             string  `json:"title"`
	StartsAt          string  `json:"startsAt"`
	PublicDescription string  `json:"publicDescription"`
	LocationDisplay   string  `json:"locationDisplay"`
	TicketAllocation  int     `json:"ticketAllocation"`
	PricingMode       string  `json:"pricingMode"`
	TicketPriceCents  int     `json:"ticketPriceCents"`
	TicketCurrency    string  `json:"ticketCurrency"`
	ReservedCount     int     `json:"reservedCount"`
	CheckedInCount    int     `json:"checkedInCount"`
	Status            string  `json:"status"`
	PublicSlug        *string `json:"publicSlug"`
	PublicURL         *string `json:"publicUrl"`
}

type eventReportDTO struct {
	ID                     string                     `json:"id"`
	EventID                string                     `json:"eventId"`
	Title                  string                     `json:"title"`
	StartsAt               string                     `json:"startsAt"`
	PublicURL              string                     `json:"publicUrl"`
	TicketAllocation       int                        `json:"ticketAllocation"`
	TicketsReserved        int                        `json:"ticketsReserved"`
	TicketsCheckedIn       int                        `json:"ticketsCheckedIn"`
	NoShows                int                        `json:"noShows"`
	SettlementSummary      *eventSettlementSummaryDTO `json:"settlementSummary,omitempty"`
	GeneratedAt            string                     `json:"generatedAt"`
	GeneratedByMemberEmail string                     `json:"generatedByMemberEmail"`
}

type eventSettlementSummaryDTO struct {
	Currency              string `json:"currency"`
	GrossPaidRevenueCents int    `json:"grossPaidRevenueCents"`
	PaidTicketCount       int    `json:"paidTicketCount"`
	PendingTicketCount    int    `json:"pendingTicketCount"`
	CancelledTicketCount  int    `json:"cancelledTicketCount"`
	FreeTicketCount       int    `json:"freeTicketCount"`
	ReservedCount         int    `json:"reservedCount"`
}

type eventRow struct {
	ID                string
	WorkspaceID       string
	Title             string
	StartsAt          time.Time
	PublicDescription string
	LocationDisplay   string
	TicketAllocation  int
	PricingMode       string
	TicketPriceCents  int
	TicketCurrency    string
	Status            string
	PublicSlug        sql.NullString
	ReservedCount     int
	CheckedInCount    int
	CreatedByPersonID string
	GeneratedAt       sql.NullTime
	GeneratedByEmail  sql.NullString
	ReportID          sql.NullString
	Snapshot          []byte
}

type createEventRequest struct {
	Title             string `json:"title"`
	StartsAt          string `json:"startsAt"`
	PublicDescription string `json:"publicDescription"`
	LocationDisplay   string `json:"locationDisplay"`
	TicketAllocation  int    `json:"ticketAllocation"`
	PricingMode       string `json:"pricingMode"`
	TicketPriceCents  int    `json:"ticketPriceCents"`
	TicketCurrency    string `json:"ticketCurrency"`
}

type updateEventRequest struct {
	Title             *string `json:"title"`
	StartsAt          *string `json:"startsAt"`
	PublicDescription *string `json:"publicDescription"`
	LocationDisplay   *string `json:"locationDisplay"`
	TicketAllocation  *int    `json:"ticketAllocation"`
	PricingMode       *string `json:"pricingMode"`
	TicketPriceCents  *int    `json:"ticketPriceCents"`
	TicketCurrency    *string `json:"ticketCurrency"`
}

func (a *App) handleListEvents(w http.ResponseWriter, r *http.Request) {
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
		select e.id, e.workspace_id, e.title, e.starts_at, e.public_description, e.location_display,
		       e.ticket_allocation, e.pricing_mode, e.ticket_price_cents, e.ticket_currency, e.status, e.public_slug,
		       (select count(*) from tickets t where t.event_id = e.id and t.payment_status <> 'cancelled') as reserved_count,
		       (select count(*) from tickets t where t.event_id = e.id and t.status = 'checked_in' and t.payment_status <> 'cancelled') as checked_in_count
		from events e
		where e.workspace_id = $1
		order by e.created_at desc, e.title
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load events")
		return
	}
	defer rows.Close()

	events := make([]eventDTO, 0)
	for rows.Next() {
		var row eventRow
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.Title, &row.StartsAt, &row.PublicDescription, &row.LocationDisplay, &row.TicketAllocation, &row.PricingMode, &row.TicketPriceCents, &row.TicketCurrency, &row.Status, &row.PublicSlug, &row.ReservedCount, &row.CheckedInCount); err != nil {
			writeError(w, http.StatusInternalServerError, "could not load events")
			return
		}
		events = append(events, a.eventDTOFromRow(row))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load events")
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func (a *App) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	title := strings.TrimSpace(req.Title)
	publicDescription := strings.TrimSpace(req.PublicDescription)
	locationDisplay := strings.TrimSpace(req.LocationDisplay)
	startsAt, err := parseRFC3339Time(req.StartsAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startsAt")
		return
	}
	if title == "" || publicDescription == "" || locationDisplay == "" {
		writeError(w, http.StatusBadRequest, "title, publicDescription, and locationDisplay are required")
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

	var eventID string
	if err := tx.QueryRow(r.Context(), `
		insert into events (workspace_id, title, starts_at, public_description, location_display, ticket_allocation, pricing_mode, ticket_price_cents, ticket_currency, status, created_by_person_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'draft', $10)
		returning id
	`, workspaceID, title, startsAt, publicDescription, locationDisplay, req.TicketAllocation, pricingMode, ticketPriceCents, ticketCurrency, actorID).Scan(&eventID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create event")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "event.created", "event", eventID, map[string]any{
		"workspaceId":       workspaceID,
		"title":             title,
		"startsAt":          startsAt.UTC().Format(time.RFC3339Nano),
		"publicDescription": publicDescription,
		"locationDisplay":   locationDisplay,
		"ticketAllocation":  req.TicketAllocation,
		"pricingMode":       pricingMode,
		"ticketPriceCents":  ticketPriceCents,
		"ticketCurrency":    ticketCurrency,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event")
		return
	}

	row := eventRow{ID: eventID, WorkspaceID: workspaceID, Title: title, StartsAt: startsAt, PublicDescription: publicDescription, LocationDisplay: locationDisplay, TicketAllocation: req.TicketAllocation, PricingMode: pricingMode, TicketPriceCents: ticketPriceCents, TicketCurrency: ticketCurrency, Status: "draft"}
	writeJSON(w, http.StatusOK, a.eventDTOFromRow(row))
}

func (a *App) handleGetEvent(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, a.eventDTOFromRow(event))
}

func (a *App) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
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
	actorID, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req updateEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Title == nil && req.StartsAt == nil && req.PublicDescription == nil && req.LocationDisplay == nil && req.TicketAllocation == nil && req.PricingMode == nil && req.TicketPriceCents == nil && req.TicketCurrency == nil {
		writeError(w, http.StatusBadRequest, "no changes provided")
		return
	}
	if event.Status == "end_of_night" {
		writeError(w, http.StatusConflict, "event is closed")
		return
	}

	newTitle := event.Title
	newStartsAt := event.StartsAt
	newPublicDescription := event.PublicDescription
	newLocationDisplay := event.LocationDisplay
	newTicketAllocation := event.TicketAllocation
	newPricingMode := event.PricingMode
	newTicketPriceCents := event.TicketPriceCents
	newTicketCurrency := event.TicketCurrency
	changed := false

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		if event.Status != "draft" && title != event.Title {
			writeError(w, http.StatusConflict, "title cannot change after publish")
			return
		}
		if title != event.Title {
			newTitle = title
			changed = true
		}
	}
	if req.PublicDescription != nil {
		value := strings.TrimSpace(*req.PublicDescription)
		if value == "" {
			writeError(w, http.StatusBadRequest, "publicDescription is required")
			return
		}
		if value != event.PublicDescription {
			newPublicDescription = value
			changed = true
		}
	}
	if req.LocationDisplay != nil {
		value := strings.TrimSpace(*req.LocationDisplay)
		if value == "" {
			writeError(w, http.StatusBadRequest, "locationDisplay is required")
			return
		}
		if value != event.LocationDisplay {
			newLocationDisplay = value
			changed = true
		}
	}
	if req.StartsAt != nil {
		value, parseErr := parseRFC3339Time(*req.StartsAt)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid startsAt")
			return
		}
		if event.Status != "draft" && !value.Equal(event.StartsAt) && event.ReservedCount > 0 {
			writeError(w, http.StatusConflict, "cannot change startsAt after reservations exist")
			return
		}
		if !value.Equal(event.StartsAt) {
			newStartsAt = value
			changed = true
		}
	}
	if req.TicketAllocation != nil {
		if *req.TicketAllocation < 0 {
			writeError(w, http.StatusBadRequest, "ticketAllocation must be non-negative")
			return
		}
		if event.Status != "draft" && *req.TicketAllocation < event.ReservedCount {
			writeError(w, http.StatusConflict, "ticket allocation cannot go below reserved tickets")
			return
		}
		if *req.TicketAllocation != event.TicketAllocation {
			newTicketAllocation = *req.TicketAllocation
			changed = true
		}
	}
	if req.PricingMode != nil || req.TicketPriceCents != nil || req.TicketCurrency != nil {
		candidateMode := event.PricingMode
		candidatePriceCents := event.TicketPriceCents
		candidateCurrency := event.TicketCurrency
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
		if normalizedMode != event.PricingMode || normalizedPriceCents != event.TicketPriceCents || normalizedCurrency != event.TicketCurrency {
			if event.ReservedCount > 0 {
				writeError(w, http.StatusConflict, "pricing cannot change after tickets exist")
				return
			}
			newPricingMode = normalizedMode
			newTicketPriceCents = normalizedPriceCents
			newTicketCurrency = normalizedCurrency
			changed = true
		}
	}
	if !changed {
		writeError(w, http.StatusBadRequest, "no changes provided")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err := tx.Exec(r.Context(), `
		update events
		set title = $2,
		    starts_at = $3,
		    public_description = $4,
		    location_display = $5,
		    ticket_allocation = $6,
		    pricing_mode = $7,
		    ticket_price_cents = $8,
		    ticket_currency = $9,
		    updated_at = now()
		where id = $1
	`, event.ID, newTitle, newStartsAt, newPublicDescription, newLocationDisplay, newTicketAllocation, newPricingMode, newTicketPriceCents, newTicketCurrency); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update event")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "event.updated", "event", event.ID, map[string]any{
		"workspaceId":       event.WorkspaceID,
		"title":             newTitle,
		"startsAt":          newStartsAt.UTC().Format(time.RFC3339Nano),
		"publicDescription": newPublicDescription,
		"locationDisplay":   newLocationDisplay,
		"ticketAllocation":  newTicketAllocation,
		"pricingMode":       newPricingMode,
		"ticketPriceCents":  newTicketPriceCents,
		"ticketCurrency":    newTicketCurrency,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event")
		return
	}

	event.Title = newTitle
	event.StartsAt = newStartsAt
	event.PublicDescription = newPublicDescription
	event.LocationDisplay = newLocationDisplay
	event.TicketAllocation = newTicketAllocation
	event.PricingMode = newPricingMode
	event.TicketPriceCents = newTicketPriceCents
	event.TicketCurrency = newTicketCurrency
	writeJSON(w, http.StatusOK, a.eventDTOFromRow(event))
}

func (a *App) handlePublishEvent(w http.ResponseWriter, r *http.Request) {
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
	actorID, role, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	_ = role
	if event.Status != "draft" {
		writeError(w, http.StatusConflict, "event is already published")
		return
	}
	if strings.TrimSpace(event.Title) == "" || event.StartsAt.IsZero() || strings.TrimSpace(event.PublicDescription) == "" || strings.TrimSpace(event.LocationDisplay) == "" {
		writeError(w, http.StatusBadRequest, "event is incomplete")
		return
	}
	if event.TicketAllocation <= 0 {
		writeError(w, http.StatusBadRequest, "ticketAllocation must be greater than zero")
		return
	}

	slug := event.ID
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err := tx.Exec(r.Context(), `
		update events
		set status = 'published',
		    public_slug = coalesce(public_slug, $2),
		    published_at = coalesce(published_at, now()),
		    updated_at = now()
		where id = $1
	`, event.ID, slug); err != nil {
		writeError(w, http.StatusInternalServerError, "could not publish event")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "event.published", "event", event.ID, map[string]any{
		"workspaceId": event.WorkspaceID,
		"publicSlug":  slug,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save publication")
		return
	}

	event.Status = "published"
	event.PublicSlug = sql.NullString{String: slug, Valid: true}
	writeJSON(w, http.StatusOK, a.eventDTOFromRow(event))
}

func (a *App) handleEndOfNight(w http.ResponseWriter, r *http.Request) {
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
	if event.Status == "draft" {
		writeError(w, http.StatusConflict, "event must be published first")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if err := tx.QueryRow(r.Context(), `
		select status
		from events
		where id = $1
		for update
	`, event.ID).Scan(&event.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "could not lock event")
		return
	}

	var existingSnapshot []byte
	var existingReportID string
	var existingGeneratedAt time.Time
	var existingGeneratedBy string
	err = tx.QueryRow(r.Context(), `
		select id, generated_at, snapshot
		from event_reports
		where event_id = $1
	`, event.ID).Scan(&existingReportID, &existingGeneratedAt, &existingSnapshot)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "could not load report")
		return
	}
	if err == nil {
		if event.Status != "end_of_night" {
			if _, err := tx.Exec(r.Context(), `
				update events
				set status = 'end_of_night', ended_at = coalesce(ended_at, now()), updated_at = now()
				where id = $1
			`, event.ID); err != nil {
				writeError(w, http.StatusInternalServerError, "could not close event")
				return
			}
			txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
			if err := a.audit(txCtx, actorID, "event.end_of_night", "event", event.ID, map[string]any{
				"workspaceId": event.WorkspaceID,
				"reportId":    existingReportID,
			}); err != nil {
				writeError(w, http.StatusInternalServerError, "could not record audit")
				return
			}
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save report")
			return
		}
		var report eventReportDTO
		if err := json.Unmarshal(existingSnapshot, &report); err != nil {
			writeError(w, http.StatusInternalServerError, "could not decode report")
			return
		}
		_ = existingGeneratedAt
		_ = existingGeneratedBy
		writeJSON(w, http.StatusOK, report)
		return
	}

	if event.Status != "published" {
		writeError(w, http.StatusConflict, "event is not published")
		return
	}

	actorEmail, err := a.loadPersonEmail(r.Context(), actorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load actor")
		return
	}
	generatedAt := time.Now().UTC()
	var grossPaidRevenueCents int64
	var paidTicketCount int64
	var pendingTicketCount int64
	var cancelledTicketCount int64
	var freeTicketCount int64
	var reservedCount int64
	var checkedInCount int64
	if err := tx.QueryRow(r.Context(), `
		select
			coalesce(sum(case when payment_status = 'paid' then amount_cents else 0 end), 0),
			count(*) filter (where payment_status = 'paid'),
			count(*) filter (where payment_status = 'pending'),
			count(*) filter (where payment_status = 'cancelled'),
			count(*) filter (where payment_status = 'free'),
			count(*) filter (where payment_status <> 'cancelled'),
			count(*) filter (where status = 'checked_in' and payment_status <> 'cancelled')
		from tickets
		where event_id = $1
	`, event.ID).Scan(&grossPaidRevenueCents, &paidTicketCount, &pendingTicketCount, &cancelledTicketCount, &freeTicketCount, &reservedCount, &checkedInCount); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load report summary")
		return
	}
	if pendingTicketCount > 0 {
		writeError(w, http.StatusConflict, "pending paid tickets must settle or expire before end of night")
		return
	}
	report := eventReportDTO{
		ID:               "",
		EventID:          event.ID,
		Title:            event.Title,
		StartsAt:         event.StartsAt.UTC().Format(time.RFC3339Nano),
		PublicURL:        a.publicEventURL(a.publicSlugValue(event)),
		TicketAllocation: event.TicketAllocation,
		TicketsReserved:  int(reservedCount),
		TicketsCheckedIn: int(checkedInCount),
		NoShows:          int(reservedCount - checkedInCount),
		SettlementSummary: &eventSettlementSummaryDTO{
			Currency:              event.TicketCurrency,
			GrossPaidRevenueCents: int(grossPaidRevenueCents),
			PaidTicketCount:       int(paidTicketCount),
			PendingTicketCount:    int(pendingTicketCount),
			CancelledTicketCount:  int(cancelledTicketCount),
			FreeTicketCount:       int(freeTicketCount),
			ReservedCount:         int(reservedCount),
		},
		GeneratedAt:            generatedAt.Format(time.RFC3339Nano),
		GeneratedByMemberEmail: actorEmail,
	}
	if report.NoShows < 0 {
		report.NoShows = 0
	}
	payload, err := json.Marshal(report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not encode report")
		return
	}
	if err := tx.QueryRow(r.Context(), `
		insert into event_reports (event_id, generated_by_person_id, snapshot)
		values ($1, $2, $3)
		returning id
	`, event.ID, actorID, payload).Scan(&report.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not store report")
		return
	}
	if payload, err = json.Marshal(report); err != nil {
		writeError(w, http.StatusInternalServerError, "could not encode report")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		update event_reports
		set snapshot = $2
		where id = $1
	`, report.ID, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "could not store report")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		update events
		set status = 'end_of_night', ended_at = coalesce(ended_at, now()), updated_at = now()
		where id = $1
	`, event.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not close event")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "event.end_of_night", "event", event.ID, map[string]any{
		"workspaceId":      event.WorkspaceID,
		"reportId":         report.ID,
		"ticketsReserved":  event.ReservedCount,
		"ticketsCheckedIn": event.CheckedInCount,
		"noShows":          report.NoShows,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save report")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (a *App) handleGetReport(w http.ResponseWriter, r *http.Request) {
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

	var snapshot []byte
	if err := a.db.QueryRow(r.Context(), `
		select snapshot
		from event_reports
		where event_id = $1
	`, event.ID).Scan(&snapshot); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "report not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load report")
		return
	}
	var report eventReportDTO
	if err := json.Unmarshal(snapshot, &report); err != nil {
		writeError(w, http.StatusInternalServerError, "could not decode report")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *App) loadEventDetails(ctx context.Context, eventID string) (eventRow, error) {
	var row eventRow
	if eventID == "" {
		return row, pgx.ErrNoRows
	}
	err := a.db.QueryRow(ctx, `
		select e.id, e.workspace_id, e.title, e.starts_at, e.public_description, e.location_display,
		       e.ticket_allocation, e.pricing_mode, e.ticket_price_cents, e.ticket_currency, e.status, e.public_slug,
		       (select count(*) from tickets t where t.event_id = e.id and t.payment_status <> 'cancelled') as reserved_count,
		       (select count(*) from tickets t where t.event_id = e.id and t.status = 'checked_in' and t.payment_status <> 'cancelled') as checked_in_count
		from events e
		where e.id = $1
	`, eventID).Scan(&row.ID, &row.WorkspaceID, &row.Title, &row.StartsAt, &row.PublicDescription, &row.LocationDisplay, &row.TicketAllocation, &row.PricingMode, &row.TicketPriceCents, &row.TicketCurrency, &row.Status, &row.PublicSlug, &row.ReservedCount, &row.CheckedInCount)
	if err != nil {
		return eventRow{}, err
	}
	return row, nil
}

func (a *App) eventDTOFromRow(row eventRow) eventDTO {
	dto := eventDTO{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Title:             row.Title,
		StartsAt:          row.StartsAt.UTC().Format(time.RFC3339Nano),
		PublicDescription: row.PublicDescription,
		LocationDisplay:   row.LocationDisplay,
		TicketAllocation:  row.TicketAllocation,
		PricingMode:       row.PricingMode,
		TicketPriceCents:  row.TicketPriceCents,
		TicketCurrency:    row.TicketCurrency,
		ReservedCount:     row.ReservedCount,
		CheckedInCount:    row.CheckedInCount,
		Status:            row.Status,
	}
	if row.PublicSlug.Valid {
		slug := row.PublicSlug.String
		dto.PublicSlug = &slug
		url := a.publicEventURL(slug)
		dto.PublicURL = &url
	}
	return dto
}

func (a *App) publicEventURL(slug string) string {
	base := strings.TrimRight(strings.TrimSpace(a.config.PublicWebURL), "/")
	if base == "" {
		return "/e/" + slug
	}
	return base + "/e/" + slug
}

func (a *App) publicSlugValue(row eventRow) string {
	if row.PublicSlug.Valid {
		return row.PublicSlug.String
	}
	return ""
}

func normalizeEventPricing(pricingMode string, ticketPriceCents int, ticketCurrency string) (string, int, string, error) {
	mode := strings.ToLower(strings.TrimSpace(pricingMode))
	currency := strings.ToLower(strings.TrimSpace(ticketCurrency))
	if mode == "" {
		mode = "free"
	}
	switch mode {
	case "free":
		if ticketPriceCents != 0 {
			return "", 0, "", fmt.Errorf("ticketPriceCents must be 0 for free events")
		}
		return "free", 0, "usd", nil
	case "fixed":
		if ticketPriceCents < 50 {
			return "", 0, "", fmt.Errorf("ticketPriceCents must be at least 50 for fixed events")
		}
		if currency != "usd" {
			return "", 0, "", fmt.Errorf("ticketCurrency must be usd for fixed events")
		}
		return "fixed", ticketPriceCents, "usd", nil
	default:
		return "", 0, "", fmt.Errorf("pricingMode must be free or fixed")
	}
}

func (a *App) loadPersonEmail(ctx context.Context, personID string) (string, error) {
	var email string
	if err := a.db.QueryRow(ctx, `select email from people where id = $1`, personID).Scan(&email); err != nil {
		return "", err
	}
	return email, nil
}

func parseRFC3339Time(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}
