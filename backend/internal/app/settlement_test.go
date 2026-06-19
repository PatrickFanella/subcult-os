package app

import (
	"testing"
	"time"
)

func TestSettlementNoShowsClamp(t *testing.T) {
	if got := settlementNoShows(5, 2); got != 3 {
		t.Fatalf("expected 3 no-shows, got %d", got)
	}
	if got := settlementNoShows(1, 4); got != 0 {
		t.Fatalf("expected clamp at 0, got %d", got)
	}
}

func TestSettlementSummaryFromCounts(t *testing.T) {
	summary := settlementSummaryFromCounts("usd", settlementTicketCounts{
		GrossPaidRevenueCents: 1500,
		PaidTicketCount:       1,
		PendingTicketCount:    0,
		CancelledTicketCount:  2,
		FreeTicketCount:       3,
		ReservedCount:         4,
	})
	if summary.Currency != "usd" || summary.GrossPaidRevenueCents != 1500 || summary.PaidTicketCount != 1 || summary.PendingTicketCount != 0 || summary.CancelledTicketCount != 2 || summary.FreeTicketCount != 3 || summary.ReservedCount != 4 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestSettlementReportFromCounts(t *testing.T) {
	report := settlementReportFromCounts("/e/night-market", eventRow{ID: "event-1", Title: "Night Market", StartsAt: time.Date(2026, 6, 18, 20, 0, 0, 0, time.UTC), TicketAllocation: 20, TicketCurrency: "usd"}, settlementTicketCounts{
		GrossPaidRevenueCents: 1500,
		PaidTicketCount:       1,
		PendingTicketCount:    0,
		CancelledTicketCount:  1,
		FreeTicketCount:       1,
		ReservedCount:         2,
		CheckedInCount:        1,
	}, time.Date(2026, 6, 18, 21, 0, 0, 0, time.UTC), "owner@example.test")
	if report.EventID != "event-1" || report.PublicURL != "/e/night-market" || report.NoShows != 1 || report.GeneratedByMemberEmail != "owner@example.test" {
		t.Fatalf("unexpected report: %#v", report)
	}
	if report.SettlementSummary == nil || report.SettlementSummary.Currency != "usd" {
		t.Fatalf("expected settlement summary: %#v", report.SettlementSummary)
	}
}

func TestSettlementNetTotalCents(t *testing.T) {
	if got := settlementNetTotalCents(1500, -250); got != 1250 {
		t.Fatalf("expected 1250, got %d", got)
	}
}
