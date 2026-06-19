package app

import "testing"

func TestTicketJourneyCapacityAndAvailability(t *testing.T) {
	tests := []struct {
		name          string
		ticketAlloc   int
		reserved      int
		pricingMode   string
		wantRemaining int
		wantFull      bool
		wantFree      bool
		wantPaid      bool
	}{
		{name: "open free event", ticketAlloc: 3, reserved: 1, pricingMode: "free", wantRemaining: 2, wantFull: false, wantFree: true, wantPaid: false},
		{name: "sold out event", ticketAlloc: 1, reserved: 2, pricingMode: "free", wantRemaining: 0, wantFull: true, wantFree: false, wantPaid: false},
		{name: "fixed price event", ticketAlloc: 2, reserved: 1, pricingMode: "fixed", wantRemaining: 1, wantFull: false, wantFree: false, wantPaid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ticketJourneyCapacity(tt.ticketAlloc, tt.reserved); got != tt.wantRemaining {
				t.Fatalf("remaining tickets = %d, want %d", got, tt.wantRemaining)
			}
			if got := ticketJourneyIsFull(tt.ticketAlloc, tt.reserved); got != tt.wantFull {
				t.Fatalf("is full = %v, want %v", got, tt.wantFull)
			}
			if got := ticketJourneyCanReservePublic(tt.pricingMode, tt.ticketAlloc, tt.reserved); got != tt.wantFree {
				t.Fatalf("can reserve public = %v, want %v", got, tt.wantFree)
			}
			if got := ticketJourneyCanCreatePaidReservation(tt.pricingMode, tt.ticketAlloc, tt.reserved); got != tt.wantPaid {
				t.Fatalf("can create paid reservation = %v, want %v", got, tt.wantPaid)
			}
		})
	}
}

func TestTicketJourneyStateAndLabels(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		paymentStatus string
		wantState     ticketJourneyState
		wantStatus    string
		wantPayment   string
	}{
		{name: "free reservation", status: "reserved", paymentStatus: "free", wantState: ticketJourneyFreeReserved, wantStatus: "Reserved", wantPayment: "Free ticket"},
		{name: "pending paid reservation", status: "reserved", paymentStatus: "pending", wantState: ticketJourneyPaymentPending, wantStatus: "Reserved", wantPayment: "Payment pending"},
		{name: "paid reservation ready", status: "reserved", paymentStatus: "paid", wantState: ticketJourneyReady, wantStatus: "Reserved", wantPayment: "Paid ticket"},
		{name: "checked in ticket", status: "checked_in", paymentStatus: "paid", wantState: ticketJourneyCheckedIn, wantStatus: "Checked in", wantPayment: "Paid ticket"},
		{name: "cancelled reservation", status: "reserved", paymentStatus: "cancelled", wantState: ticketJourneyCancelled, wantStatus: "Reserved", wantPayment: "Payment cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ticketJourneyStateForTicket(tt.status, tt.paymentStatus); got != tt.wantState {
				t.Fatalf("state = %q, want %q", got, tt.wantState)
			}
			if got := ticketJourneyStatusLabel(tt.status); got != tt.wantStatus {
				t.Fatalf("status label = %q, want %q", got, tt.wantStatus)
			}
			if got := ticketJourneyPaymentLabel(tt.paymentStatus); got != tt.wantPayment {
				t.Fatalf("payment label = %q, want %q", got, tt.wantPayment)
			}
		})
	}
}

func TestTicketJourneyDoorCheckInRules(t *testing.T) {
	if !ticketJourneyCanCheckIn("free") || !ticketJourneyCanCheckIn("paid") {
		t.Fatal("expected free and paid tickets to be eligible for check-in")
	}
	if ticketJourneyCanCheckIn("pending") || ticketJourneyCanCheckIn("cancelled") {
		t.Fatal("expected pending and cancelled tickets to be ineligible for check-in")
	}
	if !ticketJourneyIsCheckedIn("checked_in") || ticketJourneyIsCheckedIn("reserved") {
		t.Fatal("unexpected checked-in state detection")
	}
	if !ticketJourneyIsPaymentPending("pending") || ticketJourneyIsPaymentPending("paid") {
		t.Fatal("unexpected payment pending detection")
	}
}
