package app

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type publicEventSummaryDTO struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	StartsAt          string `json:"startsAt"`
	PublicDescription string `json:"publicDescription"`
	LocationDisplay   string `json:"locationDisplay"`
	PricingMode       string `json:"pricingMode"`
	TicketPriceCents  int    `json:"ticketPriceCents"`
	TicketCurrency    string `json:"ticketCurrency"`
	RemainingTickets  int    `json:"remainingTickets"`
	IsFull            bool   `json:"isFull"`
	Status            string `json:"status"`
	PublicSlug        string `json:"publicSlug"`
	PublicURL         string `json:"publicUrl"`
}

func (a *App) handleListPublicEvents(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeJSON(w, http.StatusOK, []publicEventSummaryDTO{})
		return
	}

	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if utf8.RuneCountInString(query) > 120 {
		writeError(w, http.StatusBadRequest, "search query is too long")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		select e.id, e.title, e.starts_at, e.public_description, e.location_display,
		       e.ticket_allocation, e.pricing_mode, e.ticket_price_cents, e.ticket_currency,
		       (select count(*) from tickets t where t.event_id = e.id and t.payment_status <> 'cancelled') as reserved_count,
		       e.public_slug
		from events e
		where e.status = 'published' and e.public_slug is not null
		  and (
		    $1 = ''
		    or position($1 in lower(e.title)) > 0
		    or position($1 in lower(coalesce(e.public_description, ''))) > 0
		    or position($1 in lower(coalesce(e.location_display, ''))) > 0
		  )
		order by e.starts_at asc, e.created_at asc
		limit 100
	`, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list public events")
		return
	}
	defer rows.Close()

	events := make([]publicEventSummaryDTO, 0)
	for rows.Next() {
		var row struct {
			ID, Title, PublicDescription, LocationDisplay, PricingMode, TicketCurrency, PublicSlug string
			StartsAt                                                                               time.Time
			TicketAllocation, TicketPriceCents, ReservedCount                                      int
		}
		if err := rows.Scan(&row.ID, &row.Title, &row.StartsAt, &row.PublicDescription, &row.LocationDisplay, &row.TicketAllocation, &row.PricingMode, &row.TicketPriceCents, &row.TicketCurrency, &row.ReservedCount, &row.PublicSlug); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read public event")
			return
		}

		remaining := row.TicketAllocation - row.ReservedCount
		if remaining < 0 {
			remaining = 0
		}

		events = append(events, publicEventSummaryDTO{
			ID:                row.ID,
			Title:             row.Title,
			StartsAt:          row.StartsAt.UTC().Format(time.RFC3339Nano),
			PublicDescription: row.PublicDescription,
			LocationDisplay:   row.LocationDisplay,
			PricingMode:       row.PricingMode,
			TicketPriceCents:  row.TicketPriceCents,
			TicketCurrency:    row.TicketCurrency,
			RemainingTickets:  remaining,
			IsFull:            remaining == 0,
			Status:            "published",
			PublicSlug:        row.PublicSlug,
			PublicURL:         a.publicEventURL(row.PublicSlug),
		})
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not list public events")
		return
	}

	writeJSON(w, http.StatusOK, events)
}
