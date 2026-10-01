import type { TicketDTO } from '../../domain';

export function ticketJourneyDisplayName(ticket: Pick<TicketDTO, 'displayName' | 'email'>) {
	return ticket.displayName ?? ticket.email;
}

export function formatTicketCode(code: string) {
	const chunks: string[] = [];
	for (let index = 0; index < code.length; index += 4) {
		chunks.push(code.slice(index, index + 4));
	}
	return chunks.join(' ');
}

export function ticketJourneyStatusLabel(status: TicketDTO['status']) {
	return status === 'checked_in' ? 'Checked in' : 'Reserved';
}

export function ticketJourneyStatusTone(status: TicketDTO['status']) {
	return status === 'checked_in'
		? 'border-status-success/30 bg-status-surface-success text-status-success'
		: 'border-status-warning/30 bg-status-surface-warning text-status-warning';
}

type AdmissionTicket = Pick<TicketDTO, 'status' | 'paymentStatus'>;

export function ticketPaymentAllowsAdmission(paymentStatus: TicketDTO['paymentStatus']) {
	return paymentStatus === 'free' || paymentStatus === 'paid';
}

export function ticketJourneyStatusBadge(ticket: AdmissionTicket) {
	if (!ticketPaymentAllowsAdmission(ticket.paymentStatus)) return 'Not ready for entry';
	return ticket.status === 'checked_in' ? 'Already checked in' : 'Bring to door';
}

export function ticketJourneyDoorStatusBadge(status: TicketDTO['status']) {
	return status === 'checked_in' ? 'Door ready' : 'Needs check-in';
}

export function ticketJourneyStatusCopy(ticket: AdmissionTicket, checkedInAt: string) {
	if (!ticketPaymentAllowsAdmission(ticket.paymentStatus)) return 'Payment is not confirmed. This ticket is not ready for entry.';
	return ticket.status === 'checked_in'
		? `Checked in at ${checkedInAt}`
		: 'Reserved and ready. Show the code below at the door.';
}

export function ticketJourneyDoorStatusCopy(status: TicketDTO['status'], checkedInAt: string) {
	return status === 'checked_in' ? `Checked in at ${checkedInAt}` : 'Awaiting check-in';
}

export function doorCheckInButtonLabel(checkingIn: boolean, checkedIn: boolean) {
	if (checkingIn) {
		return 'Checking in…';
	}

	return checkedIn ? 'Checked in' : 'Check in';
}

export function ticketJourneyCodeCopy(ticket: AdmissionTicket) {
	if (!ticketPaymentAllowsAdmission(ticket.paymentStatus)) return 'Keep this code for support. It does not bypass payment requirements.';
	return ticket.status === 'checked_in'
		? 'This reservation has already been scanned.'
		: 'This code is what the door team needs to check you in.';
}

export function ticketJourneyPaymentLabel(paymentStatus: TicketDTO['paymentStatus']) {
	switch (paymentStatus) {
		case 'free':
			return 'Free ticket';
		case 'pending':
			return 'Payment pending';
		case 'paid':
			return 'Paid ticket';
		case 'cancelled':
			return 'Payment cancelled';
		default:
			return 'Payment unavailable';
	}
}

export function ticketJourneyPaymentTone(paymentStatus: TicketDTO['paymentStatus']) {
	switch (paymentStatus) {
		case 'free':
		case 'paid':
			return 'border-status-success/30 bg-status-surface-success text-status-success';
		case 'pending':
			return 'border-status-warning/30 bg-status-surface-warning text-status-warning';
		case 'cancelled':
			return 'border-status-danger/30 bg-status-surface-danger text-status-danger';
		default:
			return 'border-status-warning/30 bg-status-surface-warning text-status-warning';
	}
}

export function ticketJourneyPaymentBadge(ticket: Pick<TicketDTO, 'paymentStatus' | 'amountCents' | 'currency'>) {
	switch (ticket.paymentStatus) {
		case 'paid':
			return new Intl.NumberFormat([], { style: 'currency', currency: ticket.currency.toUpperCase() }).format(ticket.amountCents / 100);
		case 'pending':
			return 'Checkout open';
		case 'cancelled':
			return 'Not valid for entry';
		case 'free':
			return 'No payment';
		default:
			return 'Unverified';
	}
}

export function ticketJourneyPaymentSummary(ticket: Pick<TicketDTO, 'paymentStatus' | 'amountCents' | 'currency'>) {
	switch (ticket.paymentStatus) {
		case 'free':
			return 'No payment needed. This is a free reservation.';
		case 'paid':
			return `Paid ${new Intl.NumberFormat([], { style: 'currency', currency: ticket.currency.toUpperCase() }).format(ticket.amountCents / 100)}.`;
		case 'pending':
			return 'Checkout may still be processing. This ticket is not final until payment completes.';
		case 'cancelled':
			return 'Payment was cancelled. This ticket is not valid for entry. Contact the organizer if you need help.';
		default:
			return 'Payment status is unavailable. Refresh before relying on this ticket for entry.';
	}
}
