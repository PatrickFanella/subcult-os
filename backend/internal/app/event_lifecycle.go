package app

import (
	"strings"
	"time"
)

const (
	eventStatusDraft      = "draft"
	eventStatusPublished  = "published"
	eventStatusEndOfNight = "end_of_night"
)

type eventLifecycle struct {
	Status            string
	Title             string
	StartsAt          time.Time
	PublicDescription string
	LocationDisplay   string
	TicketAllocation  int
	PricingMode       string
	TicketPriceCents  int
	TicketCurrency    string
	ReservedCount     int
}

func lifecycleFromEventRow(row eventRow) eventLifecycle {
	return eventLifecycle{
		Status:            row.Status,
		Title:             row.Title,
		StartsAt:          row.StartsAt,
		PublicDescription: row.PublicDescription,
		LocationDisplay:   row.LocationDisplay,
		TicketAllocation:  row.TicketAllocation,
		PricingMode:       row.PricingMode,
		TicketPriceCents:  row.TicketPriceCents,
		TicketCurrency:    row.TicketCurrency,
		ReservedCount:     row.ReservedCount,
	}
}

func (l eventLifecycle) IsDraft() bool {
	return l.Status == eventStatusDraft
}

func (l eventLifecycle) IsPublished() bool {
	return l.Status == eventStatusPublished
}

func (l eventLifecycle) IsClosed() bool {
	return l.Status == eventStatusEndOfNight
}

func (l eventLifecycle) ReadyToPublish() bool {
	return strings.TrimSpace(l.Title) != "" &&
		!l.StartsAt.IsZero() &&
		strings.TrimSpace(l.PublicDescription) != "" &&
		strings.TrimSpace(l.LocationDisplay) != ""
}

func (l eventLifecycle) HasPublishableTicketAllocation() bool {
	return l.TicketAllocation > 0
}

func (l eventLifecycle) CanPublish() bool {
	return l.IsDraft() && l.ReadyToPublish() && l.HasPublishableTicketAllocation()
}

func (l eventLifecycle) CanEndOfNight() bool {
	return !l.IsDraft()
}

func (l eventLifecycle) CanEdit() bool {
	return !l.IsClosed()
}

func (l eventLifecycle) CanChangeTitle(nextTitle string) bool {
	return l.IsDraft() || nextTitle == l.Title
}

func (l eventLifecycle) CanChangeStartsAt(nextStartsAt time.Time) bool {
	return l.IsDraft() || l.ReservedCount == 0 || nextStartsAt.Equal(l.StartsAt)
}

func (l eventLifecycle) CanChangeTicketAllocation(nextAllocation int) bool {
	return l.IsDraft() || nextAllocation >= l.ReservedCount
}

func (l eventLifecycle) CanChangePricing(nextMode string, nextTicketPriceCents int, nextTicketCurrency string) bool {
	return l.ReservedCount == 0 || (nextMode == l.PricingMode && nextTicketPriceCents == l.TicketPriceCents && nextTicketCurrency == l.TicketCurrency)
}

func eventStatusIsDraft(status string) bool {
	return status == eventStatusDraft
}

func eventStatusIsPublished(status string) bool {
	return status == eventStatusPublished
}

func eventStatusIsClosed(status string) bool {
	return status == eventStatusEndOfNight
}
