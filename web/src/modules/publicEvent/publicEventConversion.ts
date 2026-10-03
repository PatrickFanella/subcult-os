import type { PublicEventDTO, TicketReservationDTO } from '../../domain';

export function publicEventPrimaryCtaLabel(event: Pick<PublicEventDTO, 'pricingMode' | 'isFull'> | null, reserving = false) {
	if (event?.isFull) return 'Sold out';
	if (reserving) return 'Reserving…';
	return event?.pricingMode === 'fixed' ? 'Buy ticket' : 'Reserve free ticket';
}

export function publicEventConversionSummary(
	event: Pick<PublicEventDTO, 'pricingMode' | 'isFull' | 'remainingTickets' | 'ticketPriceCents' | 'ticketCurrency'> | null,
	priceLabel: string,
) {
	if (!event) return 'Email required to send the ticket. Display name optional. No account needed.';
	if (event.isFull) return 'Sold out. Ask the host about returns or future dates.';
	if (event.pricingMode === 'fixed') return `Secure checkout for ${priceLabel}. Email is required for the ticket link.`;
	return `${event.remainingTickets} ${event.remainingTickets === 1 ? 'spot remains' : 'spots remain'}. Email is required for the ticket link.`;
}

export function publicEventReservationSuccessCopy(ticket: Pick<TicketReservationDTO, 'code'>) {
	return `Ticket reserved. Save code ${ticket.code} and show it at the door.`;
}

export function publicEventRoleSectionIntro(roleCount: number | null) {
	if (roleCount === null) return 'Checking public roles for this event.';
	if (roleCount === 0) return 'No public roles are open right now.';
	return 'Applying for a role does not affect your ticket.';
}
