package app

import "strings"

type ticketJourneyState string

const (
	ticketJourneyFreeReserved   ticketJourneyState = "free_reserved"
	ticketJourneyPaymentPending ticketJourneyState = "payment_pending"
	ticketJourneyReady          ticketJourneyState = "ready"
	ticketJourneyCheckedIn      ticketJourneyState = "checked_in"
	ticketJourneyCancelled      ticketJourneyState = "cancelled"
)

func ticketJourneyCapacity(ticketAllocation, reservedCount int) int {
	remaining := ticketAllocation - reservedCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

func ticketJourneyIsFull(ticketAllocation, reservedCount int) bool {
	return ticketJourneyCapacity(ticketAllocation, reservedCount) == 0
}

func ticketJourneyCanReservePublic(pricingMode string, ticketAllocation, reservedCount int) bool {
	return strings.EqualFold(strings.TrimSpace(pricingMode), "free") && !ticketJourneyIsFull(ticketAllocation, reservedCount)
}

func ticketJourneyCanCreatePaidReservation(pricingMode string, ticketAllocation, reservedCount int) bool {
	return strings.EqualFold(strings.TrimSpace(pricingMode), "fixed") && !ticketJourneyIsFull(ticketAllocation, reservedCount)
}

func ticketJourneyCanCheckIn(paymentStatus string) bool {
	switch strings.ToLower(strings.TrimSpace(paymentStatus)) {
	case "free", "paid":
		return true
	default:
		return false
	}
}

func ticketJourneyIsPaymentPending(paymentStatus string) bool {
	return strings.EqualFold(strings.TrimSpace(paymentStatus), "pending")
}

func ticketJourneyIsCheckedIn(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "checked_in")
}

func ticketJourneyStateForTicket(status, paymentStatus string) ticketJourneyState {
	if ticketJourneyIsCheckedIn(status) {
		return ticketJourneyCheckedIn
	}

	switch strings.ToLower(strings.TrimSpace(paymentStatus)) {
	case "pending":
		return ticketJourneyPaymentPending
	case "cancelled":
		return ticketJourneyCancelled
	case "free":
		return ticketJourneyFreeReserved
	case "paid":
		return ticketJourneyReady
	default:
		return ticketJourneyReady
	}
}

func ticketJourneyStatusLabel(status string) string {
	if ticketJourneyIsCheckedIn(status) {
		return "Checked in"
	}
	return "Reserved"
}

func ticketJourneyPaymentLabel(paymentStatus string) string {
	switch strings.ToLower(strings.TrimSpace(paymentStatus)) {
	case "free":
		return "Free ticket"
	case "pending":
		return "Payment pending"
	case "paid":
		return "Paid ticket"
	case "cancelled":
		return "Payment cancelled"
	default:
		return "Reserved"
	}
}
