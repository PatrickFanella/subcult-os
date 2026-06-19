import type { PublicEventSummaryDTO } from '@/api/types';

export const discoveryLoadingCopy = 'Loading published Events…';
export const discoverySearchLabel = 'Search published Events';
export const discoverySearchPlaceholder = 'Search published Events';
export const discoveryViewEventLabel = 'View Event';

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

export function discoveryEmptyTitle(query: string) {
	return query.trim() ? 'No Events match that search yet' : 'No published Events yet';
}

export function discoveryEmptyBody(query: string) {
	const trimmed = query.trim();
	return trimmed ? `No published Events matched “${trimmed}”. Try another search.` : 'Published Events will appear here when Hosts share them.';
}

export function discoveryErrorCopy(message?: string | null) {
	return message ? `Could not load published Events: ${message}` : 'Could not load published Events.';
}
