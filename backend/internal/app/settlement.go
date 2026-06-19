package app

import "time"

type settlementTicketCounts struct {
	GrossPaidRevenueCents int64
	PaidTicketCount       int64
	PendingTicketCount    int64
	CancelledTicketCount  int64
	FreeTicketCount       int64
	ReservedCount         int64
	CheckedInCount        int64
}

func settlementNoShows(reservedCount, checkedInCount int64) int {
	noShows := reservedCount - checkedInCount
	if noShows < 0 {
		return 0
	}
	return int(noShows)
}

func settlementSummaryFromCounts(currency string, counts settlementTicketCounts) eventSettlementSummaryDTO {
	return eventSettlementSummaryDTO{
		Currency:              currency,
		GrossPaidRevenueCents: int(counts.GrossPaidRevenueCents),
		PaidTicketCount:       int(counts.PaidTicketCount),
		PendingTicketCount:    int(counts.PendingTicketCount),
		CancelledTicketCount:  int(counts.CancelledTicketCount),
		FreeTicketCount:       int(counts.FreeTicketCount),
		ReservedCount:         int(counts.ReservedCount),
	}
}

func settlementReportFromCounts(publicURL string, event eventRow, counts settlementTicketCounts, generatedAt time.Time, actorEmail string) eventReportDTO {
	return eventReportDTO{
		ID:               "",
		EventID:          event.ID,
		Title:            event.Title,
		StartsAt:         event.StartsAt.UTC().Format(time.RFC3339Nano),
		PublicURL:        publicURL,
		TicketAllocation: event.TicketAllocation,
		TicketsReserved:  int(counts.ReservedCount),
		TicketsCheckedIn: int(counts.CheckedInCount),
		NoShows:          settlementNoShows(counts.ReservedCount, counts.CheckedInCount),
		SettlementSummary: func() *eventSettlementSummaryDTO {
			summary := settlementSummaryFromCounts(event.TicketCurrency, counts)
			return &summary
		}(),
		GeneratedAt:            generatedAt.Format(time.RFC3339Nano),
		GeneratedByMemberEmail: actorEmail,
	}
}

func settlementNetTotalCents(grossPaidRevenueCents, adjustmentTotalCents int) int {
	return grossPaidRevenueCents + adjustmentTotalCents
}

func settlementAdjustmentTotalCents(adjustments []eventSettlementAdjustmentDTO) int {
	total := 0
	for _, adjustment := range adjustments {
		total += adjustment.AmountCents
	}
	return total
}
