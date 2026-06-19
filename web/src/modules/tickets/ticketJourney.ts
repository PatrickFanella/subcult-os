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
		? 'border-emerald-400/30 bg-emerald-500/10 text-emerald-50'
		: 'border-amber-300/30 bg-amber-300/10 text-amber-50';
}

export function ticketJourneyStatusBadge(status: TicketDTO['status']) {
	return status === 'checked_in' ? 'Access granted' : 'Bring to door';
}

export function ticketJourneyDoorStatusBadge(status: TicketDTO['status']) {
	return status === 'checked_in' ? 'Door ready' : 'Needs check-in';
}

export function ticketJourneyStatusCopy(status: TicketDTO['status'], checkedInAt: string) {
	return status === 'checked_in'
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

export function ticketJourneyCodeCopy(status: TicketDTO['status']) {
	return status === 'checked_in'
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
			return 'Reserved';
	}
}

export function ticketJourneyPaymentTone(paymentStatus: TicketDTO['paymentStatus']) {
	switch (paymentStatus) {
		case 'free':
		case 'paid':
			return 'border-emerald-400/30 bg-emerald-500/10 text-emerald-50';
		case 'pending':
			return 'border-amber-300/30 bg-amber-300/10 text-amber-50';
		case 'cancelled':
			return 'border-rose-400/30 bg-rose-500/10 text-rose-50';
		default:
			return 'border-amber-300/30 bg-amber-300/10 text-amber-50';
	}
}

export function ticketJourneyPaymentBadge(ticket: Pick<TicketDTO, 'paymentStatus' | 'amountCents' | 'currency'>) {
	switch (ticket.paymentStatus) {
		case 'paid':
			return new Intl.NumberFormat([], { style: 'currency', currency: ticket.currency.toUpperCase() }).format(ticket.amountCents / 100);
		case 'pending':
			return 'Checkout open';
		case 'cancelled':
			return 'Needs checkout';
		case 'free':
			return 'No payment';
		default:
			return 'No payment';
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
			return 'Payment was cancelled. Finish checkout to activate this ticket.';
		default:
			return 'No payment needed.';
	}
}
