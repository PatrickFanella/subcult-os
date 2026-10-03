import type { PublicEventDTO } from '@/api/types';

export type PublicEventConversionState = {
	pricingMode: PublicEventDTO['pricingMode'];
	isFull: PublicEventDTO['isFull'];
	remainingTickets: PublicEventDTO['remainingTickets'];
};

export function publicEventPrimaryActionLabel(event: PublicEventConversionState, reserving = false) {
	if (event.isFull) return 'Sold out';
	if (reserving) return 'Reserving…';
	return event.pricingMode === 'fixed' ? 'Buy ticket' : 'Reserve free ticket';
}

export function publicEventStickyCtaHint(event: PublicEventConversionState, priceLabel: string) {
	if (event.isFull) return 'No tickets remain for this event.';
	if (event.pricingMode === 'fixed') return `${priceLabel} · secure checkout`;
	return `${event.remainingTickets} ${event.remainingTickets === 1 ? 'spot' : 'spots'} left`;
}
