import type { PublicEventSummaryDTO } from '@/api/types';

export const discoveryLoadingCopy = 'Loading events…';
export const discoveryEmptyTitle = 'No published events yet';
export const discoveryEmptyBody = 'Published web events will appear here automatically.';
export const discoveryErrorTitle = 'Could not load events';
export const discoveryDefaultErrorBody = 'Unable to load events';
export const discoveryViewDetailsLabel = 'View Details';

export function formatDiscoveryDate(value: string) {
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { month: 'short', day: 'numeric' }).format(date);
}

export function formatDiscoveryTime(value: string) {
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat([], { timeStyle: 'short' }).format(date);
}

export function discoveryPricingLabel(event: Pick<PublicEventSummaryDTO, 'pricingMode' | 'ticketPriceCents' | 'ticketCurrency'>) {
	if (event.pricingMode === 'free') {
		return 'Free';
	}

	return new Intl.NumberFormat([], { style: 'currency', currency: event.ticketCurrency.toUpperCase() }).format(event.ticketPriceCents / 100);
}

export function discoverySubtitle(event: Pick<PublicEventSummaryDTO, 'publicDescription'>) {
	return event.publicDescription || 'Published event';
}
