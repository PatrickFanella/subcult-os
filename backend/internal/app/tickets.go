package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type publicEventDTO struct {
	eventDTO
	RemainingTickets int  `json:"remainingTickets"`
	IsFull           bool `json:"isFull"`
}

type ticketDTO struct {
	ID          string  `json:"id"`
	EventID     string  `json:"eventId"`
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName"`
	Code        string  `json:"code"`
	Status      string  `json:"status"`
	CheckedInAt *string `json:"checkedInAt"`
}

type reserveTicketRequest struct {
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName"`
}

type ticketRow struct {
	ID          string
	EventID     string
	Email       string
	DisplayName sql.NullString
	Code        string
	Status      string
	CheckedInAt sql.NullTime
}

type reservedTicketResponse struct {
	ticketDTO
	TicketURL string `json:"ticketUrl"`
}

type paidReservationResponse struct {
	TicketID          string `json:"ticketId"`
	CheckoutSessionID string `json:"checkoutSessionId"`
	CheckoutURL       string `json:"checkoutUrl"`
}

func (a *App) handlePublicEvent(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	event, err := a.loadPublishedEventBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}
	remaining := event.TicketAllocation - event.ReservedCount
	if remaining < 0 {
		remaining = 0
	}
	writeJSON(w, http.StatusOK, publicEventDTO{
		eventDTO:         a.eventDTOFromRow(event),
		RemainingTickets: remaining,
		IsFull:           remaining == 0,
	})
}

func (a *App) handleReserveTicket(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	var req reserveTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := normalizeEmail(req.Email)
	if !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	displayName := normalizeDisplayName(req.DisplayName)

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var event eventRow
	err = tx.QueryRow(r.Context(), `
		select e.id, e.workspace_id, e.title, e.starts_at, e.public_description, e.location_display,
		       e.ticket_allocation, e.pricing_mode, e.ticket_price_cents, e.ticket_currency, e.status, e.public_slug,
		       (select count(*) from tickets t where t.event_id = e.id and t.payment_status <> 'cancelled') as reserved_count,
		       (select count(*) from tickets t where t.event_id = e.id and t.status = 'checked_in' and t.payment_status <> 'cancelled') as checked_in_count
		from events e
		where e.public_slug = $1
		  and e.status = 'published'
		for update
	`, r.PathValue("slug")).Scan(&event.ID, &event.WorkspaceID, &event.Title, &event.StartsAt, &event.PublicDescription, &event.LocationDisplay, &event.TicketAllocation, &event.PricingMode, &event.TicketPriceCents, &event.TicketCurrency, &event.Status, &event.PublicSlug, &event.ReservedCount, &event.CheckedInCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}
	if event.ReservedCount >= event.TicketAllocation {
		writeError(w, http.StatusConflict, "event is full")
		return
	}
	if event.PricingMode != "free" {
		writeError(w, http.StatusConflict, "paid checkout is required for this event")
		return
	}

	code, err := newTicketCode()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create ticket code")
		return
	}
	var ticket ticketRow
	if err := tx.QueryRow(r.Context(), `
		insert into tickets (event_id, email, display_name, code, status)
		values ($1, $2, $3, $4, 'reserved')
		returning id, event_id, email, display_name, code, status, checked_in_at
	`, event.ID, email, displayName, code).Scan(&ticket.ID, &ticket.EventID, &ticket.Email, &ticket.DisplayName, &ticket.Code, &ticket.Status, &ticket.CheckedInAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create ticket")
		return
	}

	ticketURL := a.publicTicketURL(ticket.Code)
	body := fmt.Sprintf("Your ticket for %s\n\nView your ticket: %s", event.Title, ticketURL)
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, "", "ticket.reserved", "ticket", ticket.ID, map[string]any{
		"eventId":     event.ID,
		"email":       email,
		"displayName": displayName,
		"ticketUrl":   ticketURL,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := a.enqueueEmail(txCtx, email, "Your ticket for "+event.Title, body, "ticket", ticket.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not enqueue ticket email")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save ticket")
		return
	}

	writeJSON(w, http.StatusOK, reservedTicketResponse{ticketDTO: ticketDTOFromRow(ticket), TicketURL: ticketURL})
}

func (a *App) handleCreatePaidReservation(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	if a.payments == nil {
		writeError(w, http.StatusServiceUnavailable, "payment provider unavailable")
		return
	}

	var req reserveTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := normalizeEmail(req.Email)
	if !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	displayName := normalizeDisplayName(req.DisplayName)

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var event eventRow
	err = tx.QueryRow(r.Context(), `
		select e.id, e.workspace_id, e.title, e.starts_at, e.public_description, e.location_display,
		       e.ticket_allocation, e.pricing_mode, e.ticket_price_cents, e.ticket_currency, e.status, e.public_slug,
		       (select count(*) from tickets t where t.event_id = e.id and t.payment_status <> 'cancelled') as reserved_count,
		       (select count(*) from tickets t where t.event_id = e.id and t.status = 'checked_in' and t.payment_status <> 'cancelled') as checked_in_count
		from events e
		where e.public_slug = $1
		  and e.status = 'published'
		for update
	`, r.PathValue("slug")).Scan(&event.ID, &event.WorkspaceID, &event.Title, &event.StartsAt, &event.PublicDescription, &event.LocationDisplay, &event.TicketAllocation, &event.PricingMode, &event.TicketPriceCents, &event.TicketCurrency, &event.Status, &event.PublicSlug, &event.ReservedCount, &event.CheckedInCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}
	if event.PricingMode != "fixed" {
		writeError(w, http.StatusConflict, "paid checkout is only available for fixed-price events")
		return
	}
	if event.ReservedCount >= event.TicketAllocation {
		writeError(w, http.StatusConflict, "event is full")
		return
	}

	code, err := newTicketCode()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create ticket code")
		return
	}
	var ticket ticketRow
	if err := tx.QueryRow(r.Context(), `
		insert into tickets (event_id, email, display_name, code, status, payment_status, amount_cents, currency)
		values ($1, $2, $3, $4, 'reserved', 'pending', $5, $6)
		returning id, event_id, email, display_name, code, status, checked_in_at
	`, event.ID, email, displayName, code, event.TicketPriceCents, event.TicketCurrency).Scan(&ticket.ID, &ticket.EventID, &ticket.Email, &ticket.DisplayName, &ticket.Code, &ticket.Status, &ticket.CheckedInAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create ticket")
		return
	}

	checkout, err := a.payments.CreateCheckoutSession(r.Context(), checkoutSessionRequest{
		TicketID:    ticket.ID,
		EventID:     event.ID,
		EventTitle:  event.Title,
		AmountCents: event.TicketPriceCents,
		Currency:    event.TicketCurrency,
		SuccessURL:  a.publicTicketURL(ticket.Code) + "?checkout=success",
		CancelURL:   a.publicEventURL(a.publicSlugValue(event)) + "?checkout=cancelled",
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not create checkout session")
		return
	}
	if checkout.ID == "" || checkout.URL == "" {
		writeError(w, http.StatusBadGateway, "could not create checkout session")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		update tickets
		set stripe_checkout_session_id = $2
		where id = $1
	`, ticket.ID, checkout.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save checkout session")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, "", "ticket.payment_started", "ticket", ticket.ID, map[string]any{
		"eventId":               event.ID,
		"email":                 email,
		"displayName":           displayName,
		"amountCents":           event.TicketPriceCents,
		"currency":              event.TicketCurrency,
		"paymentStatus":         "pending",
		"stripeCheckoutSession": checkout.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save ticket")
		return
	}

	writeJSON(w, http.StatusOK, paidReservationResponse{TicketID: ticket.ID, CheckoutSessionID: checkout.ID, CheckoutURL: checkout.URL})
}

func (a *App) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	ticket, err := a.loadTicketByCode(r.Context(), r.PathValue("code"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load ticket")
		return
	}
	writeJSON(w, http.StatusOK, ticketDTOFromRow(ticket))
}

func (a *App) handleDoorTicketSearch(w http.ResponseWriter, r *http.Request) {
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
	_, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
	if query == "" {
		writeJSON(w, http.StatusOK, []ticketDTO{})
		return
	}

	rows, err := a.db.Query(r.Context(), `
		select id, event_id, email, display_name, code, status, checked_in_at
		from tickets
		where event_id = $1
		  and (
			lower(email) like '%' || $2 || '%'
			or lower(coalesce(display_name, '')) like '%' || $2 || '%'
			or code = $3
		  )
		order by created_at desc
	`, event.ID, query, strings.TrimSpace(r.URL.Query().Get("query")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not search tickets")
		return
	}
	defer rows.Close()

	tickets := make([]ticketDTO, 0)
	for rows.Next() {
		var ticket ticketRow
		if err := rows.Scan(&ticket.ID, &ticket.EventID, &ticket.Email, &ticket.DisplayName, &ticket.Code, &ticket.Status, &ticket.CheckedInAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not search tickets")
			return
		}
		tickets = append(tickets, ticketDTOFromRow(ticket))
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not search tickets")
		return
	}

	writeJSON(w, http.StatusOK, tickets)
}

func (a *App) handleDoorCheckIn(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var ticket ticketRow
	err = tx.QueryRow(r.Context(), `
		select id, event_id, email, display_name, code, status, checked_in_at
		from tickets
		where code = $1
		for update
	`, code).Scan(&ticket.ID, &ticket.EventID, &ticket.Email, &ticket.DisplayName, &ticket.Code, &ticket.Status, &ticket.CheckedInAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load ticket")
		return
	}
	if ticket.EventID != event.ID {
		writeError(w, http.StatusConflict, "ticket belongs to a different event")
		return
	}
	if ticket.Status == "checked_in" {
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save check-in")
			return
		}
		writeJSON(w, http.StatusOK, ticketDTOFromRow(ticket))
		return
	}

	checkedInAt := time.Now().UTC()
	if err := tx.QueryRow(r.Context(), `
		update tickets
		set status = 'checked_in',
		    checked_in_at = $2,
		    checked_in_by_person_id = $3
		where id = $1
		returning checked_in_at
	`, ticket.ID, checkedInAt, actorID).Scan(&ticket.CheckedInAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check in ticket")
		return
	}
	ticket.Status = "checked_in"
	if ticket.CheckedInAt.Valid {
		checkedInAt = ticket.CheckedInAt.Time.UTC()
	}
	ticket.CheckedInAt = sql.NullTime{Time: checkedInAt, Valid: true}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "ticket.checked_in", "ticket", ticket.ID, map[string]any{
		"eventId": event.ID,
		"code":    ticket.Code,
		"email":   ticket.Email,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save check-in")
		return
	}

	writeJSON(w, http.StatusOK, ticketDTOFromRow(ticket))
}

func (a *App) loadPublishedEventBySlug(ctx context.Context, slug string) (eventRow, error) {
	var row eventRow
	if slug == "" {
		return row, pgx.ErrNoRows
	}
	if err := a.db.QueryRow(ctx, `
		select e.id, e.workspace_id, e.title, e.starts_at, e.public_description, e.location_display,
		       e.ticket_allocation, e.pricing_mode, e.ticket_price_cents, e.ticket_currency, e.status, e.public_slug,
		       (select count(*) from tickets t where t.event_id = e.id and t.payment_status <> 'cancelled') as reserved_count,
		       (select count(*) from tickets t where t.event_id = e.id and t.status = 'checked_in' and t.payment_status <> 'cancelled') as checked_in_count
		from events e
		where e.public_slug = $1
		  and e.status = 'published'
	`, slug).Scan(&row.ID, &row.WorkspaceID, &row.Title, &row.StartsAt, &row.PublicDescription, &row.LocationDisplay, &row.TicketAllocation, &row.PricingMode, &row.TicketPriceCents, &row.TicketCurrency, &row.Status, &row.PublicSlug, &row.ReservedCount, &row.CheckedInCount); err != nil {
		return eventRow{}, err
	}
	return row, nil
}

func (a *App) loadTicketByCode(ctx context.Context, code string) (ticketRow, error) {
	var row ticketRow
	if code == "" {
		return row, pgx.ErrNoRows
	}
	if err := a.db.QueryRow(ctx, `
		select id, event_id, email, display_name, code, status, checked_in_at
		from tickets
		where code = $1
	`, code).Scan(&row.ID, &row.EventID, &row.Email, &row.DisplayName, &row.Code, &row.Status, &row.CheckedInAt); err != nil {
		return ticketRow{}, err
	}
	return row, nil
}

func (a *App) publicTicketURL(code string) string {
	base := strings.TrimRight(strings.TrimSpace(a.config.PublicWebURL), "/")
	if base == "" {
		return "/tickets/" + code
	}
	return base + "/tickets/" + code
}

func ticketDTOFromRow(row ticketRow) ticketDTO {
	dto := ticketDTO{
		ID:          row.ID,
		EventID:     row.EventID,
		Email:       row.Email,
		Code:        row.Code,
		Status:      row.Status,
		DisplayName: nil,
		CheckedInAt: nil,
	}
	if row.DisplayName.Valid {
		dto.DisplayName = &row.DisplayName.String
	}
	if row.CheckedInAt.Valid {
		value := row.CheckedInAt.Time.UTC().Format(time.RFC3339Nano)
		dto.CheckedInAt = &value
	}
	return dto
}

func newTicketCode() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
