package app

import (
	"testing"
	"time"
)

func TestEventLifecycleHelpers(t *testing.T) {
	startsAt := time.Date(2026, time.June, 18, 19, 0, 0, 0, time.UTC)
	ready := lifecycleFromEventRow(eventRow{
		Status:            eventStatusDraft,
		Title:             "Night Market",
		StartsAt:          startsAt,
		PublicDescription: "Doors open at 7",
		LocationDisplay:   "Main Hall",
		TicketAllocation:  50,
		PricingMode:       "free",
		TicketPriceCents:  0,
		TicketCurrency:    "usd",
	})
	if !ready.IsDraft() || ready.IsPublished() || ready.IsClosed() {
		t.Fatalf("unexpected lifecycle flags for draft: %#v", ready)
	}
	if !ready.ReadyToPublish() || !ready.HasPublishableTicketAllocation() || !ready.CanPublish() {
		t.Fatalf("expected draft event to be publish-ready: %#v", ready)
	}
	if !ready.CanEdit() || !ready.CanChangeTitle("Night Market") || !ready.CanChangeStartsAt(startsAt) || !ready.CanChangeTicketAllocation(50) || !ready.CanChangePricing("free", 0, "usd") {
		t.Fatalf("expected draft lifecycle to allow unchanged edits: %#v", ready)
	}
	if ready.CanEndOfNight() {
		t.Fatalf("draft event should not be able to enter end-of-night flow: %#v", ready)
	}

	published := ready
	published.Status = eventStatusPublished
	published.ReservedCount = 2
	if published.IsDraft() || !published.IsPublished() || published.IsClosed() {
		t.Fatalf("unexpected lifecycle flags for published: %#v", published)
	}
	if published.CanPublish() {
		t.Fatalf("published event should not be publishable: %#v", published)
	}
	if !published.CanEdit() || published.CanChangeTitle("After Hours") || !published.CanChangeTitle("Night Market") {
		t.Fatalf("published edit rules should only permit unchanged title: %#v", published)
	}
	if published.CanChangeStartsAt(startsAt.Add(time.Hour)) || !published.CanChangeStartsAt(startsAt) {
		t.Fatalf("published edit rules should respect reservation-gated startsAt changes: %#v", published)
	}
	if published.CanChangeTicketAllocation(1) || !published.CanChangeTicketAllocation(2) {
		t.Fatalf("published ticket allocation rules should prevent dropping below reserved count: %#v", published)
	}
	if published.CanChangePricing("fixed", 1800, "usd") || !published.CanChangePricing("free", 0, "usd") {
		t.Fatalf("published pricing rules should be bounded by existing tickets: %#v", published)
	}
	if !published.CanEndOfNight() {
		t.Fatalf("published event should be able to run end-of-night: %#v", published)
	}

	closed := ready
	closed.Status = eventStatusEndOfNight
	if closed.IsDraft() || closed.IsPublished() || !closed.IsClosed() {
		t.Fatalf("unexpected lifecycle flags for closed: %#v", closed)
	}
	if closed.CanEdit() || closed.CanPublish() || !closed.CanEndOfNight() {
		t.Fatalf("closed event should stay out of edit/publish flow but still support end-of-night reporting: %#v", closed)
	}

	incomplete := lifecycleFromEventRow(eventRow{Status: eventStatusDraft, Title: "", StartsAt: startsAt, PublicDescription: "", LocationDisplay: "  ", TicketAllocation: 0})
	if incomplete.ReadyToPublish() || incomplete.HasPublishableTicketAllocation() || incomplete.CanPublish() {
		t.Fatalf("expected incomplete event to fail publish readiness: %#v", incomplete)
	}

	if !eventStatusIsDraft(eventStatusDraft) || !eventStatusIsPublished(eventStatusPublished) || !eventStatusIsClosed(eventStatusEndOfNight) {
		t.Fatalf("expected raw status helpers to match lifecycle constants")
	}
}
